package parser

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tealeg/xlsx"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Paribas struct {
	DataExtractors map[string]DataExtractor
}

func NewParibas() *Paribas {
	return &Paribas{
		DataExtractors: map[string]DataExtractor{
			"v1": DataExtractorV1{},
			"v2": DataExtractorV2{},
		},
	}
}

func (p *Paribas) Type() database.TransactionSource {
	return database.Paribas
}

func (p *Paribas) SplitExcel(
	_ context.Context,
	data []byte,
) ([][]byte, error) {
	return [][]byte{data}, nil
}

func (p *Paribas) extractFromCellV1(cells []*xlsx.Cell) string {
	var values []string

	for i, c := range cells {
		values = append(values, c.String())

		if i > 20 { // just to ensure no extra data
			break
		}
	}

	return strings.Join(values, "-")
}

func (p *Paribas) getExtractor(cells []*xlsx.Cell) (DataExtractor, error) {
	if len(cells) < 6 {
		return nil, errors.New("row count is to short to determine the extractor type")
	}
	if cells[5].String() == "Nadawca" {
		return p.DataExtractors["v2"], nil
	}
	return p.DataExtractors["v1"], nil
}

func (p *Paribas) ParseMessages(
	ctx context.Context,
	rawArr []*Record,
) ([]*database.Transaction, error) {
	var transactions []*database.Transaction

	for _, raw := range rawArr {
		fileData, err := xlsx.OpenBinary(raw.Data)
		if err != nil {
			return nil, err
		}

		if len(fileData.Sheets) == 0 {
			return nil, errors.New("no sheets found")
		}

		sheet := fileData.Sheets[0]

		if len(sheet.Rows) < 2 {
			return nil, errors.New("no rows found")
		}

		extractor, err := p.getExtractor(sheet.Rows[0].Cells)
		if err != nil {
			return nil, err
		}

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
				OriginalMessage:             raw.Message,
				OriginalTxType:              "",
			}

			row := sheet.Rows[i]

			if len(row.Cells) < 6 {
				continue
			}

			if zeroVal := row.Cells[0].String(); strings.TrimSpace(zeroVal) == "" {
				continue // looks like empty row, skip
			}

			transactions = append(transactions, tx)

			tx.DeduplicationKeys = append(tx.DeduplicationKeys, p.extractFromCellV1(row.Cells))

			data, parseErr := extractor.Extract(ctx, row.Cells)
			if parseErr != nil {
				tx.ParsingError = parseErr
				continue
			}
			tx.Date = data.Date
			tx.DateFromMessage = data.DateFromMessage
			tx.OriginalTxType = data.TransactionType
			tx.Raw = data.Raw
			tx.Description = data.Description

			var transactionType = data.TransactionType
			var currency = data.Currency
			var transactionCurrency = data.TransactionCurrency
			var amountParsed = data.Amount
			var kwotaParsed = data.TransactionAmount
			var account = data.Account
			var destinationAccount = data.DestinationAccount
			var executedAt = data.ExecutedAt
			var amount = data.AmountString
			var kwotaStr = data.TransactionAmountString

			skipExtraChecks := false

			switch transactionType {
			case "Transakcja kartą", "Transakcja BLIK", "Prowizje i opłaty",
				"Blokada środków", "Operacja gotówkowa", "Inne operacje", "Przelew podatkowy":
				if amountParsed.GreaterThan(decimal.Zero) {
					tx.Type = database.TransactionTypeIncome
					tx.SourceAmount = amountParsed.Abs()
					tx.SourceCurrency = currency
					tx.DestinationCurrency = transactionCurrency
					tx.DestinationAmount = kwotaParsed.Abs()
					tx.DestinationAccount = account
				} else {
					tx.Type = database.TransactionTypeExpense
					tx.SourceAccount = account
					tx.SourceAmount = amountParsed.Abs()
					tx.SourceCurrency = currency
					tx.DestinationCurrency = transactionCurrency
					tx.DestinationAmount = kwotaParsed.Abs()
					skipExtraChecks = true
				}
			case "Przelew zagraniczny": // income
				if kwotaParsed.IsPositive() {
					tx.Type = database.TransactionTypeIncome
					tx.DestinationAccount = account
					tx.DestinationAmount = amountParsed.Abs()
					tx.DestinationCurrency = currency

					tx.SourceCurrency = transactionCurrency
					tx.SourceAmount = kwotaParsed.Abs()
					tx.SourceAccount = destinationAccount
				} else {
					tx.Type = database.TransactionTypeExpense
					tx.SourceAccount = account
					tx.SourceAmount = amountParsed.Abs()
					tx.SourceCurrency = currency
					tx.DestinationCurrency = transactionCurrency
					tx.DestinationAmount = kwotaParsed.Abs()
					tx.DestinationAccount = destinationAccount
				}
			case "Przelew przychodzący": // income transfer, maybe local ?
				tx.Type = database.TransactionTypeIncome // can be changed in merge
				tx.DestinationAccount = account
				tx.DestinationAmount = amountParsed.Abs()
				tx.DestinationCurrency = currency

				tx.SourceCurrency = transactionCurrency
				tx.SourceAmount = kwotaParsed.Abs()
				tx.SourceAccount = destinationAccount

				skipExtraChecks = true
			case "Przelew wychodzący", "Przelew na telefon", "Spłata karty":
				tx.Type = database.TransactionTypeRemoteTransfer // can be changed in merge
				tx.DestinationAccount = destinationAccount
				tx.DestinationAmount = amountParsed.Abs()
				tx.DestinationCurrency = currency
				tx.SourceCurrency = currency
				tx.SourceAmount = amountParsed.Abs()
				tx.SourceAccount = account
			default:
				tx.ParsingError = errors.Newf("unknown transaction type: %s", transactionType)
				continue
			}

			tx.DeduplicationKeys = append(tx.DeduplicationKeys,
				strings.Join([]string{
					tx.SourceCurrency,
					tx.DestinationCurrency,
					tx.SourceAccount,
					tx.DestinationAccount,
					tx.Date.Format("2006-01-02"),
					tx.SourceAmount.String(),
					tx.DestinationAmount.String(),
					tx.Description,
					tx.OriginalNadawcaName,
					tx.OriginalTxType,
					transactionType,
				}, "$$"),
			)

			if transactionType == "Blokada środków" && executedAt == "" {
				tx.ParsingError = errors.New("transaction is still pending. will skip from firefly for now")
				continue
			}

			if !skipExtraChecks {
				if transactionCurrency != currency {
					tx.ParsingError = errors.Newf("currency mismatch: %s != %s", transactionCurrency, currency)
					// todo find cases when different
					continue
				}

				if amount != kwotaStr {
					tx.ParsingError = errors.Newf("amount mismatch: %s != %s", amount, kwotaStr)
					// todo find cases when different
					continue
				}
			}
		}
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	merged, err := p.merge(ctx, transactions)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

func (p *Paribas) stripAccountPrefix(account string) string {
	account = strings.ToLower(account)
	if strings.HasPrefix(account, "pl") {
		account = strings.ReplaceAll(account, "pl", "")
	}

	return account
}

func (p *Paribas) merge(
	_ context.Context,
	transactions []*database.Transaction,
) ([]*database.Transaction, error) {
	var final []*database.Transaction

	var isPaymentByCard = func(tx *database.Transaction) bool {
		return tx.OriginalTxType == "Spłata karty"
	}

	for _, tx := range transactions {

		if tx.ParsingError != nil { // pass-through
			final = append(final, tx)
			continue
		}

		isDuplicate := false

		for _, f := range final {
			if tx.OriginalTxType == "Prowizje i opłaty" {
				continue
			}

			if tx.Type == database.TransactionTypeExpense {
				continue
			}

			isCreditPaymentTx := isPaymentByCard(tx)
			isCreditPaymentF := isPaymentByCard(f)

			if isCreditPaymentTx && f.Description == "Spłata karty" {
				fmt.Println("tets")
			}
			if isCreditPaymentF && tx.Description == "Spłata karty" {
				fmt.Println("tets 123")
			}

			if !(isCreditPaymentTx || isCreditPaymentF) && f.Description != tx.Description {
				continue
			}

			if isCreditPaymentTx && f.Description != "Spłata karty" {
				continue
			}

			if isCreditPaymentF && tx.Description != "Spłata karty" {
				continue
			}

			if !f.Date.Equal(tx.Date) {
				continue
			}

			if len(f.DuplicateTransactions) > 0 {
				continue // already merged
			}

			if f.SourceCurrency != "" && tx.SourceCurrency != "" && f.DestinationCurrency != "" && tx.DestinationCurrency != "" {
				if !f.SourceAmount.Equal(tx.SourceAmount) &&
					tx.DestinationCurrency == f.DestinationCurrency && tx.SourceCurrency == f.SourceCurrency {
					continue // very similar tx with same description but as amounts are different mostlikely this two separate tx
				}
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

			if tx.OriginalTxType == "Przelew przychodzący" {
				f.DestinationAmount = tx.DestinationAmount
				f.DestinationCurrency = tx.DestinationCurrency
			}

			if tx.OriginalTxType == "Przelew wychodzący" {
				f.SourceAmount = tx.SourceAmount
				f.SourceCurrency = tx.SourceCurrency
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
