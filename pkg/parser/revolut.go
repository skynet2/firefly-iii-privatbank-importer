package parser

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/common"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Revolut struct {
}

func (m *Revolut) Type() database.TransactionSource {
	return database.Revolut
}

func NewRevolut() *Revolut {
	return &Revolut{}
}

func (m *Revolut) SplitExcel(ctx context.Context, data []byte) ([][]byte, error) {
	return m.SplitCsv(ctx, data)
}

func (m *Revolut) AccountName(input string) string {
	return fmt.Sprintf("revolut_%s", input)
}

func (m *Revolut) SplitCsv(
	_ context.Context,
	data []byte,
) ([][]byte, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	linesData, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(linesData) == 0 || len(linesData) == 1 {
		return nil, errors.New("empty file")
	}

	headerIndex := 1

	var resultFiles [][]byte
	for i := headerIndex; i < len(linesData); i++ {
		targetLines := linesData[i : i+1]
		if len(targetLines) == 0 {
			break
		}

		if targetLines[0][0] == "" {
			break
		}

		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		if err = writer.WriteAll(linesData[i : i+1]); err != nil {
			return nil, err
		}

		writer.Flush()

		resultFiles = append(resultFiles, buf.Bytes())
	}

	return resultFiles, nil
}

func (m *Revolut) ParseMessages(
	ctx context.Context,
	rawArr []*Record,
) ([]*database.Transaction, error) {
	var transactions []*database.Transaction

	for _, raw := range rawArr {
		rawCsv, err := hex.DecodeString(string(raw.Data))
		if err != nil {
			return nil, err
		}

		reader := csv.NewReader(bytes.NewReader(rawCsv))
		reader.FieldsPerRecord = -1

		tx := &database.Transaction{
			ID:                uuid.NewString(),
			Raw:               string(rawCsv),
			OriginalMessage:   raw.Message,
			TransactionSource: m.Type(),
		}
		transactions = append(transactions, tx)

		linesData, err := reader.ReadAll()
		if err != nil {
			tx.ParsingError = err
			continue
		}

		additionalTx, parsingErr := m.parseTransaction(tx, linesData[0])
		if parsingErr != nil {
			tx.ParsingError = parsingErr
			continue
		}

		transactions = append(transactions, additionalTx...)
	}

	return m.adjustAmount(m.merge(ctx, transactions)), nil
}

func (m *Revolut) adjustAmount(
	transactions []*database.Transaction,
) []*database.Transaction {
	for _, tx := range transactions {
		tx.SourceAmount = tx.SourceAmount.Abs()
		tx.DestinationAmount = tx.DestinationAmount.Abs()
	}

	return transactions
}

func (m *Revolut) merge(
	_ context.Context,
	transactions []*database.Transaction,
) []*database.Transaction {
	var finalTransactions []*database.Transaction
	var duplicates []*database.Transaction

	for _, tx := range transactions {
		if tx.Type != database.TransactionTypeInternalTransfer {
			finalTransactions = append(finalTransactions, tx)
			continue
		}

		if lo.Contains(duplicates, tx) {
			continue
		}

		for _, t := range transactions {
			if t == tx || t.Type != database.TransactionTypeInternalTransfer {
				continue
			}

			if lo.Contains(duplicates, t) {
				continue
			}

			if lo.Contains(finalTransactions, t) {
				continue
			}

			if t.Description == tx.Description && t.Date.Equal(tx.Date) {
				tx.DuplicateTransactions = append(tx.DuplicateTransactions, t)

				if tx.SourceAmount.LessThan(decimal.Zero) {
					tx.DestinationAmount = t.DestinationAmount
					tx.DestinationCurrency = t.DestinationCurrency
					tx.DestinationAccount = t.DestinationAccount
				} else {
					tx.SourceAmount = t.SourceAmount
					tx.SourceCurrency = t.SourceCurrency
					tx.SourceAccount = t.SourceAccount
				}

				finalTransactions = append(finalTransactions, tx)
				break
			}
		}
	}

	return finalTransactions
}

func (m *Revolut) parseTransaction(
	tx *database.Transaction,
	data []string,
) ([]*database.Transaction, error) {
	if len(data) < 8 {
		return nil, errors.Newf("expected len > 8, got %d", len(data))
	}

	invisibleChars := strings.TrimFunc(data[2], func(r rune) bool {
		return !unicode.IsGraphic(r)
	})

	operationType := data[0]

	operationTime, timeErr := time.Parse("2006-01-02 15:04:05", invisibleChars)
	if timeErr != nil {
		return nil, errors.Wrapf(timeErr, "failed to parse operation time %s", data[2])
	}

	sourceAmount, err := decimal.NewFromString(data[5])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse source amount %s", data[5])
	}

	supportedStates := []string{
		"COMPLETED",
		"PENDING",
	}

	state := data[8]

	if !lo.Contains(supportedStates, state) {
		return nil, errors.Newf("unsupported state %s", state)
	}

	tx.Type = database.TransactionTypeExpense
	tx.Date = operationTime

	tx.SourceAmount = sourceAmount
	tx.SourceCurrency = data[7]
	tx.SourceAccount = m.AccountName(tx.SourceCurrency)

	tx.Description = fmt.Sprintf("%s.%s", operationType, data[4])

	tx.DeduplicationKeys = []string{
		strings.Join([]string{
			operationType,
			data[2],
			data[4],
			data[5],
			data[7],
		}, "_"),
	}

	if operationType == "EXCHANGE" {
		tx.Type = database.TransactionTypeInternalTransfer

		if sourceAmount.GreaterThan(decimal.Zero) { // it destination
			tx.DestinationCurrency = tx.SourceCurrency
			tx.DestinationAmount = sourceAmount.Abs()
			tx.DestinationAccount = m.AccountName(tx.DestinationCurrency)

			tx.SourceCurrency = ""
			tx.SourceAmount = decimal.Zero
			tx.SourceAccount = ""
		}

		return nil, nil
	}

	tx.SourceAmount = sourceAmount.Abs()

	if sourceAmount.GreaterThan(decimal.Zero) {
		return nil, errors.WithStack(common.ErrOperationNotSupported)
	}

	return nil, nil
}
