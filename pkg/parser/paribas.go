package parser

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/tealeg/xlsx"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Paribas struct {
}

func NewParibas() *Paribas {
	return &Paribas{}
}

func (p *Paribas) ParseMessages(
	ctx context.Context,
	raw []byte,
	_ time.Time,
) ([]*database.Transaction, error) {
	fileData, err := xlsx.OpenBinary(raw)
	if err != nil {
		return nil, err
	}

	if len(fileData.Sheets) == 0 {
		return nil, nil
	}

	var transactions []*database.Transaction

	sheet := fileData.Sheets[0]

	for i := 1; i < len(sheet.Rows); i++ {
		tx := &database.Transaction{
			ID:                          uuid.NewString(),
			Type:                        0,
			SourceAmount:                decimal.Decimal{},
			SourceCurrency:              "",
			DestinationAmount:           decimal.Decimal{},
			DestinationCurrency:         "",
			Date:                        time.Time{},
			Description:                 "",
			SourceAccount:               "",
			DestinationAccount:          "",
			DateFromMessage:             "",
			Raw:                         "",
			InternalTransferDirectionTo: false,
			DuplicateTransactions:       nil,
			OriginalMessage:             nil,
		}
		transactions = append(transactions, tx)

		row := sheet.Rows[i]

		if len(row.Cells) < 6 {
			continue
		}

		date, cellErr := row.Cells[0].GetTime(false)
		if cellErr != nil {
			return nil, cellErr
		}

		tx.Date = date
		tx.DateFromMessage = date.Format("15:04")

		amount := row.Cells[3].String()
		amountParsed, amountErr := decimal.NewFromString(amount)
		if amountErr != nil {
			return nil, errors.Join(amountErr, errors.Newf("can not parse amount: %s", amount))
		}

		currency := row.Cells[4].String()
		senderOrReceiver := row.Cells[5].String()
		description := row.Cells[6].String()

		rawAccount := row.Cells[7].String()
		accountArr := toLines(strings.ToLower(rawAccount))
		account := lo.Reverse(accountArr)[0]

		transactionType := row.Cells[8].String()

		kwota := row.Cells[9].String()
		transactionCurrency := row.Cells[10].String()
		//status := row.Cells[11].String()

		if transactionCurrency != currency {
			// tood find cases when different
			return nil, errors.Newf("currency mismatch: %s != %s", transactionCurrency, currency)
		}

		if amount != kwota {
			// tood find cases when different
			return nil, errors.Newf("amount mismatch: %s != %s", amount, kwota)
		}

		tx.Raw = strings.Join([]string{description, senderOrReceiver, rawAccount, transactionType}, "\n")
		tx.Description = description
		switch transactionType {
		case "Transakcja kartą", "Transakcja BLIK", "Prowizje i opłaty":
			tx.Type = database.TransactionTypeExpense
			tx.SourceAccount = account
			tx.SourceAmount = amountParsed.Abs()
			tx.SourceCurrency = currency
		case "Przelew zagraniczny": // income
			tx.Type = database.TransactionTypeIncome
			tx.DestinationAccount = account
			tx.DestinationAmount = amountParsed.Abs()
			tx.DestinationCurrency = currency
		case "Przelew przychodzący": // income transfer, maybe local ?
			tx.Type = database.TransactionTypeIncome // can be changed in merge
			tx.DestinationAccount = account
			tx.DestinationAmount = amountParsed.Abs()
			tx.DestinationCurrency = currency
		case "Przelew wychodzący":
			tx.Type = database.TransactionTypeRemoteTransfer // can be changed in merge
			tx.DestinationAccount = toLines(senderOrReceiver)[0]
			tx.DestinationAmount = amountParsed.Abs()
			tx.DestinationCurrency = currency
			tx.SourceCurrency = currency
			tx.SourceAmount = amountParsed.Abs()
			tx.SourceAccount = account
		default:
			return nil, errors.Newf("unknown transaction type: %s", transactionType)
		}
	}

	merged, err := p.merge(ctx, transactions)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

//var currencyExchangeRegex = regexp.MustCompile(`(\w{3}) (\w{3}) ([^ ]+) (.*)$`)

func (p *Paribas) merge(
	_ context.Context,
	transactions []*database.Transaction,
) ([]*database.Transaction, error) {
	var final []*database.Transaction

	for _, tx := range transactions {

		isDuplicate := false
		for _, f := range final {
			if f.Description != tx.Description {
				continue
			}

			if !f.Date.Equal(tx.Date) {
				continue
			}

			if len(f.DuplicateTransactions) > 0 {
				continue // already merged
			}

			if f.SourceCurrency == "" {
				f.SourceCurrency = tx.SourceCurrency
			}
			if f.DestinationCurrency == "" {
				f.DestinationCurrency = tx.DestinationCurrency
			}
			if f.SourceAmount.IsZero() {
				f.SourceAmount = tx.SourceAmount
			}
			if f.DestinationAmount.IsZero() {
				f.DestinationAmount = tx.DestinationAmount
			}
			if f.SourceAccount == "" {
				f.SourceAccount = tx.SourceAccount
			}
			if f.DestinationAccount == "" {
				f.DestinationAccount = tx.DestinationAccount
			}

			//isCurrencyExchange := len(currencyExchangeRegex.FindStringSubmatch(tx.Description)) == 5 // USD PLN 4.0006 TWM2131232132131
			f.Type = database.TransactionTypeInternalTransfer
			tx.Type = database.TransactionTypeInternalTransfer

			isDuplicate = true
			f.DuplicateTransactions = append(f.DuplicateTransactions, tx)
			break
		}

		if isDuplicate {
			continue
		}

		final = append(final, tx)
	}

	return final, nil
}
