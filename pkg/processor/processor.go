package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Processor struct {
	repo            Repo
	parser          Parser
	notificationSvc NotificationSvc
}

func NewProcessor(
	repo Repo,
	parser Parser,
	notificationSvc NotificationSvc,
) *Processor {
	return &Processor{
		repo:            repo,
		parser:          parser,
		notificationSvc: notificationSvc,
	}
}

func (p *Processor) ProcessMessage(
	ctx context.Context,
	message Message,
) error {
	lower := strings.ToLower(message.Content)

	switch lower {
	case "dry":
		return p.DryRun(ctx, message)
	case "clear":
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
		Content:     message.Content,
	})
	if err != nil {
		return err
	}

	if err = p.notificationSvc.React(ctx, message.ChatID, message.MessageID); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("failed to react to message")
	}

	return nil
}

func (p *Processor) Clear(
	ctx context.Context,
) error {
	return p.repo.Clear(ctx)
}

func (p *Processor) DryRun(ctx context.Context, message Message) error {
	transaction, errArr, err := p.ProcessLatestMessages(ctx)
	if err != nil {
		return err
	}

	fmt.Println(transaction, errArr)

	return nil
}

func (p *Processor) ProcessLatestMessages(
	ctx context.Context,
) ([]*database.Transaction, []error, error) {
	messages, err := p.repo.GetLatestMessages(ctx)
	if err != nil {
		return nil, nil, err
	}

	var transactions []*database.Transaction
	var parseErrorsArr []error

	for _, message := range messages {
		transaction, parserErr := p.parser.ParseMessages(ctx, message.Content, message.CreatedAt)
		if err != nil {
			parseErrorsArr = append(parseErrorsArr, errors.Join(
				errors.Wrapf(parserErr, "message: %s", message.Content)))

			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, parseErrorsArr, nil
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

			if !f.Amount.Equal(tx.Amount) || f.DateFromMessage != tx.DateFromMessage {
				continue // not our tx
			}

			if tx.InternalTransferDirectionTo && f.InternalTransferDirectionTo {
				continue // not our tx
			}

			if tx.InternalTransferDirectionTo {
				if tx.DestinationAccount != f.SourceAccount {
					continue // not our tx
				}
			} else {
				if tx.SourceAccount != f.DestinationAccount {
					continue // not our tx
				}
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
