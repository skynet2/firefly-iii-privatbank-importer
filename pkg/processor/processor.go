package processor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
)

const (
	reactionAccepted  = "🤝"
	reactionCommitted = "🍾"
	failedToCommit    = "🤬"
)

type Processor struct {
	repo            Repo
	parser          Parser
	notificationSvc NotificationSvc
	fireflySvc      Firefly
}

func NewProcessor(
	repo Repo,
	parser Parser,
	notificationSvc NotificationSvc,
	fireflySvc Firefly,
) *Processor {
	return &Processor{
		repo:            repo,
		fireflySvc:      fireflySvc,
		parser:          parser,
		notificationSvc: notificationSvc,
	}
}

func (p *Processor) ProcessMessage(
	ctx context.Context,
	message Message,
) error {
	lower := strings.ToLower(message.Content)

	trimmed := strings.Split(lower, "@")
	switch trimmed[0] {
	case "/dry":
		return p.DryRun(ctx, message)
	case "/commit":
		return p.Commit(ctx, message)
	case "/clear":
		return p.Clear(ctx)
	default:
		return p.AddMessage(ctx, message)
	}
}

func (p *Processor) AddMessage(
	ctx context.Context,
	message Message,
) error {
	err := p.repo.AddMessage(ctx, database.Message{
		ID:          uuid.NewString(),
		CreatedAt:   message.OriginalDate,
		ProcessedAt: nil,
		IsProcessed: false,
		Content:     message.Content,
		ChatID:      message.ChatID,
		MessageID:   message.MessageID,
	})
	if err != nil {
		return err
	}

	if err = p.notificationSvc.React(ctx, message.ChatID, message.MessageID, reactionAccepted); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to react to message")
	}

	return nil
}

func (p *Processor) Clear(
	ctx context.Context,
) error {
	return p.repo.Clear(ctx)
}

func (p *Processor) prettyPrint(
	ctx context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
	err error,
	message Message,
) error {
	var sb strings.Builder
	for _, tx := range mappedTx {
		if tx.IsCommitted {
			sb.WriteString("Committed: ✅\n")
		}
		if tx.MappingError != nil {
			sb.WriteString("Has Error: ❌\n")
		}
		sb.WriteString(fmt.Sprintf("Date: %s\n", tx.Original.Date.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("\nSource: %v%v", tx.Original.SourceAmount.StringFixed(2), tx.Original.SourceCurrency))
		sb.WriteString(fmt.Sprintf("\nSource Account: %s", tx.Original.SourceAccount))
		if tx.Transaction != nil {
			sb.WriteString(fmt.Sprintf("\nSource [FF]: %s", tx.Transaction.SourceName))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("\nDestination: %v%v",
			tx.Original.DestinationAmount.StringFixed(2), tx.Original.DestinationCurrency))
		sb.WriteString(fmt.Sprintf("\nDestination Account: %s", tx.Original.DestinationAccount))
		if tx.Transaction != nil {
			sb.WriteString(fmt.Sprintf("\nDestination [FF]: %s", tx.Transaction.DestinationName))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("\nType: %v", tx.Original.Type))
		if tx.Transaction != nil {
			sb.WriteString(fmt.Sprintf("\nType [FF]: %s", tx.Transaction.Type))
		}
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("\nDescription: %s", tx.Original.Description))

		if tx.MappingError != nil {
			sb.WriteString(fmt.Sprintf("\nERROR: %s", tx.MappingError))
		}
		sb.WriteString("\n====================\n")
	}

	if len(errArr) > 0 {
		sb.WriteString("\n\nErrors:\n")
		for _, err = range errArr {
			sb.WriteString(fmt.Sprintf("%s\n", err))
		}
	}

	if err = p.notificationSvc.SendMessage(ctx, message.ChatID, sb.String()); err != nil {
		return err
	}

	return nil
}

func (p *Processor) DryRun(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	if err = p.prettyPrint(ctx, mappedTx, errArr, err, message); err != nil {
		p.SendErrorMessage(ctx, err, message)
	}

	return nil
}

func (p *Processor) ProcessLatestMessages(
	ctx context.Context,
) ([]*firefly.MappedTransaction, []error, error) {
	messages, err := p.repo.GetLatestMessages(ctx)
	if err != nil {
		return nil, nil, err
	}

	var transactions []*database.Transaction
	var parseErrorsArr []error

	for _, message := range messages {
		transaction, parserErr := p.parser.ParseMessages(ctx, message.Content, message.CreatedAt)
		if parserErr != nil {
			parseErrorsArr = append(parseErrorsArr, errors.Join(
				errors.Wrapf(parserErr, "message: %s", message.Content)))

			continue
		}

		transaction.OriginalMessage = message
		transactions = append(transactions, transaction)
	}

	transactions, err = p.Merge(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	mappedTransactions, err := p.fireflySvc.MapTransactions(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	return mappedTransactions, parseErrorsArr, nil
}

func (p *Processor) Merge(
	_ context.Context,
	messages []*database.Transaction,
) ([]*database.Transaction, error) {
	var finalTransactions []*database.Transaction

	for _, tx := range messages {
		if tx.Type != database.TransactionTypeInternalTransfer {
			finalTransactions = append(finalTransactions, tx)
			continue
		}

		// currently we have a transfer transaction, lets ensure that we dont have duplicates
		isDuplicate := false
		for _, f := range finalTransactions {
			if f.Type != database.TransactionTypeInternalTransfer {
				continue
			}

			if f.DateFromMessage != tx.DateFromMessage {
				continue // not our tx
			}

			if tx.InternalTransferDirectionTo && f.InternalTransferDirectionTo {
				continue // not our tx
			}

			if tx.DestinationAccount != f.DestinationAccount ||
				tx.SourceAccount != f.SourceAccount {
				continue
			}

			if f.DestinationCurrency == "" && tx.DestinationCurrency != "" {
				f.DestinationCurrency = tx.DestinationCurrency
			}
			if f.SourceCurrency == "" && tx.SourceCurrency != "" {
				f.SourceCurrency = tx.SourceCurrency
			}

			if f.DestinationAmount.Equal(decimal.Zero) && tx.DestinationAmount.GreaterThan(decimal.Zero) {
				f.DestinationAmount = tx.DestinationAmount
			}
			if f.SourceAmount.Equal(decimal.Zero) && tx.SourceAmount.GreaterThan(decimal.Zero) {
				f.SourceAmount = tx.SourceAmount
			}

			// otherwise we have a duplicate
			f.DuplicateTransactions = append(f.DuplicateTransactions, tx)
			isDuplicate = true
		}

		if isDuplicate {
			continue
		}

		finalTransactions = append(finalTransactions, tx)
	}

	return finalTransactions, nil
}

func (p *Processor) Commit(ctx context.Context, message Message) error {
	transactions, errArr, err := p.ProcessLatestMessages(ctx)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)
		return err
	}

	for _, tx := range transactions {
		p.CommitTransaction(ctx, tx, message)
	}

	if err = p.prettyPrint(ctx, transactions, errArr, err, message); err != nil {
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
		transaction.MappingError = errors.Join(transaction.MappingError,
			errors.Newf("original message is nil"))

		return
	}

	transaction.IsCommitted = true
	if _, err := p.fireflySvc.CreateTransactions(ctx, transaction.Transaction); err != nil {
		transaction.MappingError = errors.Join(transaction.MappingError,
			errors.Wrapf(err, "failed to commit transaction"))
	}

	reaction := reactionCommitted
	if transaction.MappingError != nil {
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

	if transaction.MappingError != nil {
		return
	}

	tt := time.Now().UTC()

	for _, upd := range toUpdate {
		upd.ProcessedAt = &tt
		upd.IsProcessed = true

		if err := p.repo.UpdateMessage(ctx, transaction.Original.OriginalMessage); err != nil {
			transaction.MappingError = errors.Join(transaction.MappingError,
				errors.Wrapf(err, "failed to update message"))

			p.SendErrorMessage(ctx, transaction.MappingError, requestMessage)
		}
	}
}

func (p *Processor) SendErrorMessage(ctx context.Context, err error, message Message) {
	if err = p.notificationSvc.SendMessage(ctx, message.ChatID,
		fmt.Sprintf("Failed to process command: %v\n Error: %v", message.Content, err)); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to send message")
	}
}
