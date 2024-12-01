package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gammazero/workerpool"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/common"
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
	cfg *Config
}

type Config struct {
	Repo             Repo
	Parsers          map[database.TransactionSource]Parser
	NotificationSvc  NotificationSvc
	FireflySvc       Firefly
	DuplicateCleaner DuplicateCleaner
	Printer          Printer
}

func NewProcessor(
	cfg *Config,
) *Processor {
	return &Processor{
		cfg: cfg,
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
	var err error

	trimmed := strings.Split(lower, "@")
	switch trimmed[0] {
	case "/dry":
		err = p.DryRun(ctx, message)
	case "/stat":
		err = p.Stat(ctx, message)
	case "/duplicates":
		err = p.Duplicates(ctx, message)
	case "/errors":
		err = p.Errors(ctx, message)
	case "/commit":
		err = p.Commit(ctx, message)
	case "/clear":
		err = p.Clear(ctx, message)
	default:
		err = p.AddMessage(ctx, message)
	}

	if err != nil {
		p.SendErrorMessage(ctx, err, message)
		return nil
	}

	return err
}

func (p *Processor) AddMessage(
	ctx context.Context,
	message Message,
) error {
	var targetMessages []database.Message

	if message.FileID != "" {
		fileData, fileErr := p.cfg.NotificationSvc.GetFile(ctx, message.FileID)
		if fileErr != nil {
			return errors.Wrapf(fileErr, "failed to get file")
		}

		splitted, err := p.cfg.Parsers[message.TransactionSource].SplitExcel(ctx, fileData)
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

	err := p.cfg.Repo.AddMessage(ctx, targetMessages)
	if err != nil {
		return err
	}

	if err = p.cfg.NotificationSvc.React(ctx, message.ChatID, message.MessageID, reactionAccepted); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to react to message")
	}

	return nil
}

func (p *Processor) Clear(ctx context.Context, message Message) error {
	return p.cfg.Repo.Clear(ctx, message.TransactionSource)
}

func (p *Processor) DryRun(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	return p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID, p.cfg.Printer.Dry(ctx, mappedTx, errArr))
}

func (p *Processor) Stat(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	return p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID, p.cfg.Printer.Stat(ctx, mappedTx, errArr))
}

func (p *Processor) Errors(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	return p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID, p.cfg.Printer.Errors(ctx, mappedTx, errArr))
}

func (p *Processor) Duplicates(ctx context.Context, message Message) error {
	mappedTx, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)

		return nil
	}

	return p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID, p.cfg.Printer.Duplicates(ctx, mappedTx, errArr))
}

func (p *Processor) ProcessLatestMessages(
	ctx context.Context,
	transactionSource database.TransactionSource,
) ([]*firefly.MappedTransaction, []error, error) {
	messages, err := p.cfg.Repo.GetLatestMessages(ctx, transactionSource)
	if err != nil {
		return nil, nil, err
	}

	var parseErrorsArr []error

	parser, ok := p.cfg.Parsers[transactionSource]
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

	mappedTransactions, err := p.cfg.FireflySvc.MapTransactions(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	for _, tx := range mappedTransactions {
		if tx.Error != nil {
			continue
		}

		if err = p.cfg.DuplicateCleaner.IsDuplicate(ctx, tx.Original.DeduplicationKey, transactionSource); err != nil {
			tx.Error = errors.Join(tx.Error, err)
		}
	}

	return mappedTransactions, parseErrorsArr, nil
}

func (p *Processor) Commit(ctx context.Context, message Message) error {
	transactions, errArr, err := p.ProcessLatestMessages(ctx, message.TransactionSource)
	if err != nil {
		p.SendErrorMessage(ctx, err, message)
		return err
	}

	pool := workerpool.New(5)
	var commitResults []*CommitResult
	var mut sync.Mutex

	var messagesToUpdate []*database.Message

	for _, tx := range transactions {
		if errors.Is(tx.Error, common.ErrDuplicate) { // do not commit duplicates, but mark them
			tx.Original.OriginalMessage.IsProcessed = true
			tx.Original.OriginalMessage.ProcessedAt = lo.ToPtr(time.Now().UTC())
			messagesToUpdate = append(messagesToUpdate, tx.Original.OriginalMessage)
			continue
		}

		if tx.Error != nil {
			continue
		}

		txCopy := tx
		pool.Submit(func() {
			res := p.CommitTransaction(ctx, txCopy, message)
			mut.Lock()
			commitResults = append(commitResults, res...)
			mut.Unlock()
		})
	}

	pool.StopWait()

	var finalErr error

	for _, tx := range commitResults {
		if tx.Msg.IsProcessed {
			messagesToUpdate = append(messagesToUpdate, tx.Msg)
		}

		if tx.Tx != nil && tx.Tx.Original != nil {
			extractedDuplicationKeys := p.ExtractDuplicationKeys(tx.Tx.Original)

			for _, key := range extractedDuplicationKeys {
				finalErr = errors.Join(finalErr,
					p.cfg.DuplicateCleaner.AddDuplicateKey(ctx, key, tx.Msg.TransactionSource),
				)
			}
		}
	}

	if err = p.cfg.Repo.UpdateMessages(ctx, messagesToUpdate); err != nil {
		return err
	}

	updatedMessages := map[int64]struct{}{}

	for _, upd := range commitResults {
		if _, ok := updatedMessages[upd.Msg.MessageID]; ok {
			continue
		}

		if notifyErr := p.cfg.NotificationSvc.React(ctx,
			upd.Msg.ChatID,
			upd.Msg.MessageID,
			upd.ExpectedReaction,
		); notifyErr != nil {
			zerolog.Ctx(ctx).Error().Err(notifyErr).Msg("failed to react to message")
		}

		updatedMessages[upd.Msg.MessageID] = struct{}{}
	}

	return p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID, p.cfg.Printer.Commit(ctx, transactions, errArr))
}

func (p *Processor) ExtractDuplicationKeys(tx *database.Transaction) []string {
	if tx == nil {
		return nil
	}

	var keys []string

	if tx.DeduplicationKey != "" {
		keys = append(keys, tx.DeduplicationKey)
	}

	for _, dup := range tx.DuplicateTransactions {
		if dup == nil {
			continue
		}

		if dup == tx {
			continue
		}

		if dupIDs := p.ExtractDuplicationKeys(dup); len(dupIDs) > 0 {
			for _, k := range dupIDs {
				if !lo.Contains(keys, k) {
					keys = append(keys, dupIDs...)
				}
			}
		}

	}

	return keys
}

func (p *Processor) CommitTransaction(
	ctx context.Context,
	transaction *firefly.MappedTransaction,
	_ Message,
) []*CommitResult {
	if transaction.Original.OriginalMessage == nil {
		transaction.Error = errors.Join(transaction.Error, errors.Newf("original message is nil"))

		return nil
	}

	if _, err := p.cfg.FireflySvc.CreateTransactions(
		ctx,
		transaction.Transaction,
		transaction.Original.DeduplicationKey != "",
	); err != nil {
		transaction.Error = errors.Join(transaction.Error, errors.Wrapf(err, "failed to commit transaction"))
	}

	reaction := reactionCommitted

	if transaction.Error != nil {
		reaction = failedToCommit
	} else {
		transaction.IsCommitted = true
	}

	toUpdate := []*CommitResult{
		{
			Tx:               transaction,
			Msg:              transaction.Original.OriginalMessage,
			ExpectedReaction: reaction,
		},
	}

	for _, tx := range transaction.Original.DuplicateTransactions {
		toUpdate = append(toUpdate, &CommitResult{
			ExpectedReaction: reaction,
			Tx:               transaction,
			Msg:              tx.OriginalMessage,
		})
	}

	now := time.Now().UTC()

	for _, upd := range toUpdate {
		upd.Msg.ProcessedAt = &now
		upd.Msg.IsProcessed = true
	}

	return toUpdate
}

func (p *Processor) SendErrorMessage(ctx context.Context, err error, message Message) {
	if err = p.cfg.NotificationSvc.SendMessage(ctx, message.ChatID,
		fmt.Sprintf("Failed to process command: %v\n Error: %v", message.Content, err)); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to send message")
	}
}
