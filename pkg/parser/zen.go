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
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Zen struct {
}

func (z *Zen) SplitExcel(ctx context.Context, data []byte) ([][]byte, error) {
	return z.SplitCsv(ctx, data)
}

func NewZen() *Zen {
	return &Zen{}
}

func (z *Zen) AccountName(input string) string {
	return fmt.Sprintf("zen_%s", input)
}

func (z *Zen) Type() database.TransactionSource {
	return database.Zen
}

func (z *Zen) SplitCsv(
	_ context.Context,
	data []byte,
) ([][]byte, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	linesData, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var headerIndex int
	for index, line := range linesData {
		if len(line) == 0 {
			continue
		}

		if line[0] == "Date" {
			headerIndex = index
			break
		}
	}

	if headerIndex == 0 {
		return nil, errors.New("header not found")
	}

	headerIndex += 1

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

func (z *Zen) ParseMessages(
	_ context.Context,
	rawArr []*Record,
) ([]*database.Transaction, error) {
	var transactions []*database.Transaction

	for _, raw := range rawArr {
		rawCsv, err := hex.DecodeString(string(raw.Data))
		if err != nil {
			return nil, err
		}

		reader := csv.NewReader(bytes.NewReader(rawCsv))

		tx := &database.Transaction{
			ID:  uuid.NewString(),
			Raw: string(raw.Data),
		}
		transactions = append(transactions, tx)

		linesData, err := reader.ReadAll()
		if err != nil {
			tx.ParsingError = err
			continue
		}

		if len(linesData) != 1 {
			tx.ParsingError = errors.Newf("expected length 1, got %d", len(linesData[0]))
			continue
		}

		if len(linesData) == 0 || len(linesData[0]) == 0 || linesData[0][0] == "" {
			break
		}

		additionalTx, parsingErr := z.parseTransaction(tx, linesData[0])
		if parsingErr != nil {
			tx.ParsingError = parsingErr
			continue
		}

		transactions = append(transactions, additionalTx...)
	}

	return transactions, nil
}

func (z *Zen) parseTransaction(
	tx *database.Transaction,
	data []string,
) ([]*database.Transaction, error) {
	if len(data) < 7 {
		return nil, errors.Newf("expected at least 7 fields, got %d. data : %v",
			len(data), spew.Sdump(data))
	}

	originalAmount, err := decimal.NewFromString(data[5])
	if err != nil {
		return nil, err
	}
	originalCurrency := data[6]

	txType := data[1]

	invisibleChars := strings.TrimFunc(data[0], func(r rune) bool {
		return !unicode.IsGraphic(r)
	})

	txData, err := time.Parse("_2-Jan-06", invisibleChars)
	if err != nil {
		return nil, err
	}

	tx.Date = txData

	settlementCurrency := data[4]
	settlementAmount, err := decimal.NewFromString(data[3])
	if err != nil {
		return nil, err
	}

	tx.Raw = strings.Join(data, ",")
	tx.Description = data[2]

	switch txType {
	case "Exchange money":
		tx.Type = database.TransactionTypeInternalTransfer
		tx.SourceAmount = settlementAmount
		tx.SourceCurrency = settlementCurrency
		tx.SourceAccount = z.AccountName(settlementCurrency)

		tx.DestinationCurrency = originalCurrency
		tx.DestinationAmount = originalAmount
		tx.DestinationAccount = z.AccountName(originalCurrency)
	default:
		if settlementAmount.LessThan(decimal.Zero) {
			tx.Type = database.TransactionTypeExpense
			tx.SourceAmount = originalAmount
			tx.SourceCurrency = originalCurrency
			tx.SourceAccount = z.AccountName(originalCurrency)
		} else {
			tx.Type = database.TransactionTypeIncome

			tx.DestinationAccount = z.AccountName(originalCurrency)
			tx.DestinationAmount = originalAmount
			tx.DestinationCurrency = originalCurrency
		}
	}

	var additionalTx []*database.Transaction
	if tx.Type == database.TransactionTypeIncome && !originalAmount.Equal(settlementAmount) { // create expense for diff
		diffAmount := originalAmount.Sub(settlementAmount)

		diffTx := &database.Transaction{
			ID:             uuid.NewString(),
			Type:           database.TransactionTypeExpense,
			SourceAmount:   diffAmount,
			SourceCurrency: originalCurrency,
			SourceAccount:  z.AccountName(originalCurrency),
			Date:           tx.Date,
			Description: fmt.Sprintf("settlement diff for %v. and original desc %v",
				tx.ID,
				tx.Description,
			),
		}

		additionalTx = append(additionalTx, diffTx)
	}

	return additionalTx, nil
}
