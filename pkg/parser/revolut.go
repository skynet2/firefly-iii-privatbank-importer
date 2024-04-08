package parser

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Revolut struct {
}

func NewRevolut() *Revolut {
	return &Revolut{}
}

func (p *Revolut) Type() database.TransactionSource {
	return database.Revolut
}

func (p *Revolut) ParseMessages(
	ctx context.Context,
	rawArr []*Record,
) ([]*database.Transaction, error) {
	var finalTx []*database.Transaction

	for _, rawItem := range rawArr {
		raw := string(rawItem.Data)
		var parsed revolutTransaction

		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			finalTx = appendTxOrError(finalTx, nil, err, string(rawItem.Data), rawItem)
			continue
		}

		switch parsed.Type {
		case "TRANSFER":
			tx, err := p.parseTransfer(parsed, raw, rawItem)
			if tx != nil {
				finalTx = append(finalTx, tx)
			}
			if err != nil {
				finalTx = appendTxOrError(finalTx, nil, err, string(rawItem.Data), rawItem)
				continue
			}
		default:
			return nil, errors.Newf("unknown transaction type: %s", parsed.Type)
		}
	}

	mergedTxs, err := p.merge(ctx, finalTx)
	if err != nil {
		return nil, err
	}

	return mergedTxs, nil
}

var (
	revolutSimpleTransferRegex = regexp.MustCompile(`To (.*)$`)
)

func (p *Revolut) merge(
	_ context.Context,
	transactions []*database.Transaction,
) ([]*database.Transaction, error) {
	var finalTx []*database.Transaction
	txMap := map[string]*database.Transaction{}

	for _, tx := range transactions {
		if tx.OriginalNadawcaName == "savings" {
			existing, ok := txMap[tx.ID] // savings have same id for both two tx on each account

			if !ok {
				txMap[tx.ID] = tx
				finalTx = append(finalTx, tx)
				continue
			}

			existing.DuplicateTransactions = append(existing.DuplicateTransactions, tx)
			continue
		}
	}

	return finalTx, nil
}

func (p *Revolut) parseTransfer(
	parsedTx revolutTransaction,
	raw string,
	item *Record,
) (*database.Transaction, error) {
	amount := decimal.NewFromInt(int64(parsedTx.Amount)).Div(decimal.NewFromInt(100))
	finalTx := &database.Transaction{
		ID:                          parsedTx.Id.String(),
		TransactionSource:           p.Type(),
		Type:                        database.TransactionTypeRemoteTransfer,
		SourceAmount:                decimal.Decimal{},
		SourceCurrency:              parsedTx.Currency,
		DestinationAmount:           decimal.Decimal{},
		DestinationCurrency:         parsedTx.Currency,
		Date:                        parsedTx.StartedAt(),
		Description:                 parsedTx.Description,
		SourceAccount:               parsedTx.Account.ID,
		DestinationAccount:          "",
		DateFromMessage:             parsedTx.StartedAt().String(),
		Raw:                         raw,
		InternalTransferDirectionTo: false,
		OriginalMessage:             item.Message,
		ParsingError:                nil,
		OriginalTxType:              parsedTx.Type,
		OriginalNadawcaName:         parsedTx.Tag,
	}

	if parsedTx.Description == "Withdrawing savings" && parsedTx.Tag == "savings" {
		finalTx.Type = database.TransactionTypeInternalTransfer

		finalTx.SourceCurrency = parsedTx.Currency
		finalTx.SourceAmount = amount.Abs()

		finalTx.DestinationCurrency = parsedTx.Currency
		finalTx.DestinationAmount = amount.Abs()

		if amount.GreaterThan(decimal.Zero) {
			finalTx.SourceAccount = parsedTx.Account.ID
			finalTx.DestinationAccount = parsedTx.Recipient.Account.ID
		} else {
			finalTx.DestinationAccount = parsedTx.Account.ID
			finalTx.SourceAccount = parsedTx.Sender.Account.ID
			finalTx.InternalTransferDirectionTo = true
		}

		return finalTx, nil
	}

	if parsedTx.Description == "Depositing savings" && parsedTx.Tag == "savings" {
		finalTx.Type = database.TransactionTypeInternalTransfer

		finalTx.SourceCurrency = parsedTx.Currency
		finalTx.SourceAmount = amount.Abs()

		finalTx.DestinationCurrency = parsedTx.Currency
		finalTx.DestinationAmount = amount.Abs()

		if amount.GreaterThan(decimal.Zero) {
			finalTx.SourceAccount = parsedTx.Sender.Account.ID
			finalTx.DestinationAccount = parsedTx.Account.ID
			finalTx.InternalTransferDirectionTo = true
		} else {
			finalTx.DestinationAccount = parsedTx.Recipient.Account.ID
			finalTx.SourceAccount = parsedTx.Account.ID
		}

		return finalTx, nil
	}

	// todo most likely multi currency transfers ?? wtf

	matches := simpleExpenseRegex.FindStringSubmatch(parsedTx.Description)
	if len(matches) != 2 {
		return nil, errors.Newf("expected 2 matches, got %v", spew.Sdump(matches))
	}

	finalTx.DestinationAccount = matches[1]

	return finalTx, nil
}
