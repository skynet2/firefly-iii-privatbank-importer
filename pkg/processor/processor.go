package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	parser2 "github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

const (
	reactionAccepted  = "🤝"
	reactionCommitted = "🍾"
	failedToCommit    = "🤬"
)

type Processor struct {
	repo            Repo
	parsers         map[database.TransactionSource]Parser
	notificationSvc NotificationSvc
	fireflySvc      Firefly
}

func NewProcessor(
	repo Repo,
	parsers []Parser,
	notificationSvc NotificationSvc,
	fireflySvc Firefly,
) *Processor {
	parser := make(map[database.TransactionSource]Parser)
	for _, p := range parsers {
		parser[p.Type()] = p
	}

	return &Processor{
		repo:            repo,
		fireflySvc:      fireflySvc,
		parsers:         parser,
		notificationSvc: notificationSvc,
	}
}

func (p *Processor) ProcessMessage(
	ctx context.Context,
	message Message,
) error {
	if message.TransactionSource == "" {
		p.SendErrorMessage(ctx, errors.New("transaction source is not set"), message)
		return nil
	}

	lower := strings.ToLower(message.Content)

	trimmed := strings.Split(lower, "@")
	switch trimmed[0] {
	case "/dry":
		return p.DryRun(ctx, message)
	case "/commit":
		return p.Commit(ctx, message)
	case "/clear":
		return p.Clear(ctx, message)
	default:
		if err := p.AddMessage(ctx, message); err != nil {
			p.SendErrorMessage(ctx, err, message)

			return err
		}

		return nil
	}
}

func (p *Processor) AddMessage(
	ctx context.Context,
	message Message,
) error {
	var targetMessages []database.Message

	if message.FileID != "" {
		fileData, fileErr := p.notificationSvc.GetFile(ctx, message.FileID)
		if fileErr != nil {
			return errors.Wrapf(fileErr, "failed to get file")
		}

		splitted, err := p.parsers[message.TransactionSource].SplitExcel(ctx, fileData)
		if err != nil {
			return errors.Wrapf(err, "failed to split file")
		}

		for _, s := range splitted {
			targetMessages = append(targetMessages, database.Message{
				ID:                uuid.NewString(),
				CreatedAt:         message.OriginalDate,
				ProcessedAt:       nil,
				IsProcessed:       false,
				Content:           hex.EncodeToString(s),
				FileID:            message.FileID,
				ChatID:            message.ChatID,
				MessageID:         message.MessageID,
				TransactionSource: message.TransactionSource,
			})
		}

	} else {
		targetMessages = append(targetMessages, database.Message{
			ID:                uuid.NewString(),
			CreatedAt:         message.OriginalDate,
			ProcessedAt:       nil,
			IsProcessed:       false,
			Content:           message.Content,
			FileID:            message.FileID,
			ChatID:            message.ChatID,
			MessageID:         message.MessageID,
			TransactionSource: message.TransactionSource,
		})
	}

	for _, m := range targetMessages {
		err := p.repo.AddMessage(ctx, m)
		if err != nil {
			return err
		}
	}

	if err := p.notificationSvc.React(ctx, message.ChatID, message.MessageID, reactionAccepted); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to react to message")
	}

	return nil
}

func (p *Processor) Clear(ctx context.Context, message Message) error {
	return p.repo.Clear(ctx, message.TransactionSource)
}

func (p *Processor) prettyPrint(
	ctx context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
	message Message,
) error {
	if len(mappedTx) == 0 && len(errArr) == 0 {
		if err := p.notificationSvc.SendMessage(ctx, message.ChatID, "No messages to process"); err != nil {
			return err
		}

		return nil
	}
	var sb strings.Builder
	withErrors := 0
	for _, tx := range mappedTx {
		if tx.IsCommitted {
			sb.WriteString("Committed: ✅\n")
		}
		if tx.FireflyMappingError != nil || tx.Original.ParsingError != nil {
			sb.WriteString("Has Error: ❌\n")
			withErrors += 1
		}

		sb.WriteString(fmt.Sprintf("Source: %v", tx.Original.TransactionSource))
		sb.WriteString(fmt.Sprintf("\nDate: %s\n", tx.Original.Date.Format("2006-01-02 15:04")))

		if !tx.Original.SourceAmount.IsZero() {
			sb.WriteString(fmt.Sprintf("\nSource: %v%v", tx.Original.SourceAmount.StringFixed(2), tx.Original.SourceCurrency))
		}
		if tx.Original.SourceAccount != "" {
			sb.WriteString(fmt.Sprintf("\nSource Account: %s", tx.Original.SourceAccount))
		}
		if tx.Transaction != nil && tx.Transaction.SourceName != "" {
			sb.WriteString(fmt.Sprintf("\nSource [FF]: %s", tx.Transaction.SourceName))
		}
		sb.WriteString("\n")

		if !tx.Original.DestinationAmount.IsZero() {
			sb.WriteString(fmt.Sprintf("\nDestination: %v%v",
				tx.Original.DestinationAmount.StringFixed(2), tx.Original.DestinationCurrency))
		}
		if tx.Original.DestinationAccount != "" {
			sb.WriteString(fmt.Sprintf("\nDestination Account: %s", tx.Original.DestinationAccount))
		}
		if tx.Transaction != nil && tx.Transaction.DestinationName != "" {
			sb.WriteString(fmt.Sprintf("\nDestination [FF]: %s", tx.Transaction.DestinationName))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("\nType: %v", tx.Original.Type))
		if tx.Transaction != nil {
			sb.WriteString(fmt.Sprintf("\nType [FF]: %s", tx.Transaction.Type))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("\nDescription: %s", tx.Original.Description))

		if tx.Original.ParsingError != nil {
			sb.WriteString(fmt.Sprintf("\nParsing ERROR: %s", tx.Original.ParsingError))
		}
		if tx.FireflyMappingError != nil {
			sb.WriteString(fmt.Sprintf("\nFirefly ERROR: %s", tx.FireflyMappingError))
		}
		sb.WriteString("\n====================\n")
	}

	if len(errArr) > 0 {
		sb.WriteString("\n\nErrors:\n")
		for _, err := range errArr {
			sb.WriteString(fmt.Sprintf("%s\n", err))
		}
	}

	sb.WriteString(fmt.Sprintf("\nTotal: %v", len(mappedTx)))
	if withErrors > 0 {
		sb.WriteString(fmt.Sprintf("\nOk: %v 🔥", len(mappedTx)-withErrors))
		sb.WriteString(fmt.Sprintf("\nErrors: %v 🚒", withErrors))
	} else {
		sb.WriteString("\nAll Ok: ✅")
	}

	if err := p.notificationSvc.SendMessage(ctx, message.ChatID, sb.String()); err != nil {
		return err
	}

	return nil
}

func (p *Processor) DryRun(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	if err = p.prettyPrint(ctx, mappedTx, errArr, message); err != nil {
		p.SendErrorMessage(ctx, err, message)
	}

	return nil
}

func (p *Processor) ProcessLatestMessages(
	ctx context.Context,
	transactionSource database.TransactionSource,
) ([]*firefly.MappedTransaction, []error, error) {
	messages, err := p.repo.GetLatestMessages(ctx, transactionSource)
	if err != nil {
		return nil, nil, err
	}

	var parseErrorsArr []error

	parser, ok := p.parsers[transactionSource]
	if !ok {
		return nil, nil, errors.Newf("parser for source %v not found", transactionSource)
	}

	var dataToProcess []*parser2.Record
	for _, message := range messages {
		rec := &parser2.Record{
			Message: message,
			Data:    []byte(message.Content),
		}

		if message.TransactionSource == database.Paribas {
			rec.Data, err = hex.DecodeString(message.Content)
			if err != nil {
				parseErrorsArr = append(parseErrorsArr, errors.Wrapf(err, "failed to decode hex"))
				continue
			}
		}

		dataToProcess = append(dataToProcess, rec)
	}

	transactions, parserErr := parser.ParseMessages(ctx, dataToProcess)
	if parserErr != nil {
		return nil, nil, parserErr
	}

	mappedTransactions, err := p.fireflySvc.MapTransactions(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	return mappedTransactions, parseErrorsArr, nil
}

func (p *Processor) Commit(ctx context.Context, message Message) error {
	transactions, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)
		return err
	}

	for _, tx := range transactions {
		p.CommitTransaction(ctx, tx, message)
	}

	if err = p.prettyPrint(ctx, transactions, errArr, message); err != nil {
		p.SendErrorMessage(ctx, err, message)
	}

	return nil
}

func (p *Processor) CommitTransaction(
	ctx context.Context,
	transaction *firefly.MappedTransaction,
	requestMessage Message,
) {
	if transaction.Original.OriginalMessage == nil {
		transaction.FireflyMappingError = errors.Join(transaction.FireflyMappingError,
			errors.Newf("original message is nil"))

		return
	}

	transaction.IsCommitted = true
	if _, err := p.fireflySvc.CreateTransactions(ctx, transaction.Transaction); err != nil {
		transaction.FireflyMappingError = errors.Join(transaction.FireflyMappingError,
			errors.Wrapf(err, "failed to commit transaction"))
	}

	reaction := reactionCommitted
	if transaction.FireflyMappingError != nil {
		reaction = failedToCommit
	}

	toUpdate := []*database.Message{
		transaction.Original.OriginalMessage,
	}
	for _, tx := range transaction.Original.DuplicateTransactions {
		toUpdate = append(toUpdate, tx.OriginalMessage)
	}

	for _, upd := range toUpdate {
		if err := p.notificationSvc.React(ctx,
			upd.ChatID,
			upd.MessageID,
			reaction,
		); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("failed to react to message")
		}
	}

	if transaction.FireflyMappingError != nil {
		return
	}

	tt := time.Now().UTC()

	for _, upd := range toUpdate {
		upd.ProcessedAt = &tt
		upd.IsProcessed = true

		if err := p.repo.UpdateMessage(ctx, upd); err != nil {
			transaction.FireflyMappingError = errors.Join(transaction.FireflyMappingError,
				errors.Wrapf(err, "failed to update message"))

			p.SendErrorMessage(ctx, transaction.FireflyMappingError, requestMessage)
		}
	}
}

func (p *Processor) SendErrorMessage(ctx context.Context, err error, message Message) {
	if err = p.notificationSvc.SendMessage(ctx, message.ChatID,
		fmt.Sprintf("Failed to process command: %v\n Error: %v", message.Content, err)); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to send message")
	}
}
