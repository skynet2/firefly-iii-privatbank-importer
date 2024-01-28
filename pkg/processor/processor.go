package processor

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Processor struct {
	repo   Repo
	parser Parser
}

func NewProcessor(
	repo Repo,
	parser Parser,
) *Processor {
	return &Processor{
		repo:   repo,
		parser: parser,
	}
}

func (p *Processor) ProcessMessage(
	ctx context.Context,
	message string,
) ([]*database.Transaction, []error, error) {
	lower := strings.ToLower(message)

	switch lower {
	case "dry":
		return p.ProcessLatestMessages(ctx)
	}
	return nil, nil, nil
}

func (p *Processor) AddMessage(
	ctx context.Context,
	message string,
) {
	p.repo.AddMessage(ctx, message)
}

func (p *Processor) Clear(
	ctx context.Context,
) error {
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
