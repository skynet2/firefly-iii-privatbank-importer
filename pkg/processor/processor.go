package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
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
	transactions, errArr, err := p.ProcessLatestMessages(ctx)
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, tx := range transactions {
		ffSource := ""
		ffDest := ""
		if tx.FireflyTransaction != nil {
			ffSource = tx.FireflyTransaction.SourceName
			ffDest = tx.FireflyTransaction.DestinationName
		}

		sb.WriteString(fmt.Sprintf("%s %s %s", tx.Amount, tx.Currency,
			tx.Date.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("\nSource %s || %s", tx.SourceAccount, ffSource))
		sb.WriteString(fmt.Sprintf("\nDestination %s || %s", tx.DestinationAccount, ffDest))
		sb.WriteString(fmt.Sprintf("\nDescription: %s", tx.Description))
		sb.WriteString(fmt.Sprintf("\nType: %s", tx.Type))

		if tx.FireflyMappingError != nil {
			sb.WriteString(fmt.Sprintf("\nERROR: %s", tx.FireflyMappingError))
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

	transactions, err = p.Merge(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	transactions, err = p.Mapper(ctx, transactions)
	if err != nil {
		return nil, nil, err
	}

	return transactions, parseErrorsArr, nil
}

func (p *Processor) Mapper(
	ctx context.Context,
	transactions []*database.Transaction,
) ([]*database.Transaction, error) {
	accounts, err := p.fireflySvc.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	accountByAccountNumber := map[string]*firefly.Account{}
	for _, acc := range accounts {
		accountByAccountNumber[acc.Attributes.AccountNumber] = acc
	}

	for _, tx := range transactions {
		switch tx.Type {
		case database.TransactionTypeRemoteTransfer:
			fallthrough
		case database.TransactionTypeExpense:
			acc, ok := accountByAccountNumber[tx.SourceAccount]
			if !ok {
				tx.FireflyMappingError = errors.Newf("account with IBAN %s not found", tx.SourceAccount)
				continue
			}

			tx.FireflyTransaction = &database.FireflyTransaction{
				Type:        "withdrawal",
				SourceID:    acc.Id,
				SourceName:  acc.Attributes.Name,
				Description: tx.Description,
				Notes:       tx.Description,
			}
		case database.TransactionTypeInternalTransfer:
			sourceID := tx.SourceAccount
			destinationID := tx.DestinationAccount

			accSource, ok := accountByAccountNumber[sourceID]
			if !ok {
				tx.FireflyMappingError = errors.Newf("source account with IBAN %s not found", sourceID)
				continue
			}

			accDestination, ok := accountByAccountNumber[destinationID]
			if !ok {
				tx.FireflyMappingError = errors.Newf("destination account with IBAN %s not found", destinationID)
				continue
			}

			tx.FireflyTransaction = &database.FireflyTransaction{
				Type:            "transfer",
				SourceID:        accSource.Id,
				SourceName:      accSource.Attributes.Name,
				DestinationID:   accDestination.Id,
				DestinationName: accDestination.Attributes.Name,
				Description:     tx.Description,
				Notes:           tx.Description,
			}
		case database.TransactionTypeIncome:
			acc, ok := accountByAccountNumber[tx.DestinationAccount]
			if !ok {
				tx.FireflyMappingError = errors.Newf("account with IBAN %s not found", tx.DestinationAccount)
				continue
			}

			tx.FireflyTransaction = &database.FireflyTransaction{
				Type:            "income",
				DestinationID:   acc.Id,
				DestinationName: acc.Attributes.Name,
				Description:     tx.Description,
				Notes:           tx.Description,
			}
		default:
			tx.FireflyMappingError = errors.Newf("unknown transaction type %d", tx.Type)
		}
	}

	return transactions, nil
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

			if tx.DestinationAccount != f.DestinationAccount ||
				tx.SourceAccount != f.SourceAccount {
				continue
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
