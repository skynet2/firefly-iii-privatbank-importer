package parser

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	simpleExpenseLinesCount  = 3
	remoteTransferLinesCount = 3
	incomeTransferLinesCount = 3
	partialRefundLinesCount  = 2
	creditPaymentLinesCount  = 2
)

const (
	unk                = "UNK"
	incomeTransferText = "зарахування переказу з картки через приват24"
)

type Parser struct {
}

func (p *Parser) SplitExcel(_ context.Context, data []byte) ([][]byte, error) {
	return nil, errors.New("not supported")
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Type() database.TransactionSource {
	return database.PrivatBank
}

func (p *Parser) ParseMessages(
	ctx context.Context,
	rawArr []*Record,
) ([]*database.Transaction, error) {
	var finalTx []*database.Transaction

	for _, rawItem := range rawArr {
		raw := string(rawItem.Data)
		lower := strings.ToLower(raw)
		lines := toLines(lower)

		if len(lines) == 0 {
			finalTx = append(finalTx, &database.Transaction{
				Raw:             raw,
				OriginalMessage: rawItem.Message,
				ParsingError:    errors.New("empty input"),
			})

			continue
		}

		if strings.HasSuffix(lines[0], "переказ зі своєї карти") { // external transfer to another bank
			remote, err := p.ParseRemoteTransfer(ctx, raw, rawItem.Message.CreatedAt)
			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		if strings.Contains(lower, "переказ на свою карт") ||
			strings.Contains(lower, "переказ зі своєї карт") { // internal transfer
			remote, err := p.ParseInternalTransfer(ctx, raw, rawItem.Message.CreatedAt)

			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		if strings.Contains(lower, "переказ через ") || strings.HasSuffix(lines[0], incomeTransferText) { // remote transfer
			if strings.Contains(lower, "відправник:") || strings.HasSuffix(lines[0], incomeTransferText) { // income
				remote, err := p.ParseIncomeTransfer(ctx, raw, rawItem.Message.CreatedAt)

				finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
				continue
			}

			remote, err := p.ParseRemoteTransfer(ctx, raw, rawItem.Message.CreatedAt)

			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		if strings.HasSuffix(lines[0], "зарахування переказу на картку") || strings.Contains(lower, "повернення.") ||
			strings.HasSuffix(lines[0], "зарахування переказу через приват24 зі своєї картки") ||
			strings.Contains(lines[0], "зарахування переказу.") {
			remote, err := p.ParseIncomingCardTransfer(ctx, raw, rawItem.Message.CreatedAt)

			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		if strings.HasSuffix(lines[0], "зарахування") {
			remote, err := p.ParsePartialRefund(ctx, raw, rawItem.Message.CreatedAt)

			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		if len(lines) == 2 && strings.HasSuffix(lines[0], " списання") {
			remote, err := p.ParseCreditPayment(ctx, raw, rawItem.Message.CreatedAt)

			finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
			continue
		}

		remote, err := p.ParseSimpleExpense(ctx, raw, rawItem.Message.CreatedAt)

		finalTx = p.appendTxOrError(finalTx, remote, err, raw, rawItem)
		continue
	}

	merged, err := p.Merge(ctx, finalTx)
	if err != nil {
		return nil, err
	}

	return merged, nil
}

func (p *Parser) Merge(
	_ context.Context,
	messages []*database.Transaction,
) ([]*database.Transaction, error) {
	var finalTransactions []*database.Transaction

	for _, tx := range messages {
		// currently we have a transfer transaction, lets ensure that we dont have duplicates
		isDuplicate := false
		for _, f := range finalTransactions {
			if f.DateFromMessage == "" || tx.DateFromMessage == "" {
				continue // missing date, can not merge
			}

			fDate, err := time.Parse("15:04", f.DateFromMessage)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			txDate, err := time.Parse("15:04", tx.DateFromMessage)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			minDiff := math.Abs(fDate.Sub(txDate).Minutes())
			if minDiff > 5 { // diff > 5m
				continue // not our tx
			}

			if tx.InternalTransferDirectionTo && f.InternalTransferDirectionTo {
				continue // not our tx
			}

			if tx.SourceAccount == f.SourceAccount {
				if tx.DestinationAccount == unk {
					tx.DestinationAccount = f.DestinationAccount
				}
				if f.DestinationAccount == unk {
					f.DestinationAccount = tx.DestinationAccount
				}
			}

			if tx.SourceAccount == "" &&
				tx.Description == "Зарахування переказу через Приват24 зі своєї картки" &&
				f.Description == "Переказ на свою карту через Приват24" {

				tx.SourceAccount = f.SourceAccount
				tx.SourceAmount = f.SourceAmount
				tx.SourceCurrency = f.SourceCurrency

				f.DestinationAccount = tx.DestinationAccount
				f.DestinationAmount = tx.DestinationAmount
				f.DestinationCurrency = tx.DestinationCurrency

				f.Type = database.TransactionTypeInternalTransfer
				tx.Type = database.TransactionTypeInternalTransfer
			}

			if f.SourceAccount == "" && // revers of the above
				f.Description == "Зарахування переказу через Приват24 зі своєї картки" &&
				tx.Description == "Переказ на свою карту через Приват24" {
				f.SourceAccount = tx.SourceAccount
				f.SourceAmount = tx.SourceAmount
				f.SourceCurrency = tx.SourceCurrency

				tx.DestinationAccount = f.DestinationAccount
				tx.DestinationAmount = f.DestinationAmount
				tx.DestinationCurrency = f.DestinationCurrency

				f.Type = database.TransactionTypeInternalTransfer
				tx.Type = database.TransactionTypeInternalTransfer
			}

			if tx.DestinationAccount != "" &&
				tx.Description == "Зарахування переказу на картку" &&
				f.DestinationAccount == "" && f.Description == "Переказ зі своєї карти" &&
				f.SourceAccount != "" {
				f.DestinationAccount = tx.DestinationAccount
				tx.SourceAccount = f.SourceAccount

				f.Type = database.TransactionTypeInternalTransfer
				tx.Type = database.TransactionTypeInternalTransfer
			}

			if tx.DestinationAccount == "" &&
				tx.Description == "Переказ зі своєї карти" &&
				f.DestinationAccount != "" &&
				f.Description == "Зарахування переказу на картку" {
				tx.DestinationAccount = f.DestinationAccount
				f.SourceAccount = tx.SourceAccount

				f.Type = database.TransactionTypeInternalTransfer
				tx.Type = database.TransactionTypeInternalTransfer
			}

			if f.Type != database.TransactionTypeInternalTransfer && tx.Type != database.TransactionTypeInternalTransfer {
				continue
			}

			// privat fuck yourself. sometime card does not have first digit
			if strings.HasPrefix(tx.Description, "Переказ зі своєї картки") &&
				strings.HasPrefix(tx.SourceAccount, "*") &&
				strings.HasPrefix(f.Description, "Переказ на свою картку") &&
				strings.HasPrefix(f.DestinationAccount, "*") {
				tx.SourceAccount = f.SourceAccount
				f.DestinationAccount = tx.DestinationAccount
			}

			// reverse previous
			if strings.HasPrefix(f.Description, "Переказ зі своєї картки") &&
				strings.HasPrefix(f.SourceAccount, "*") &&
				strings.HasPrefix(tx.Description, "Переказ на свою картку") &&
				strings.HasPrefix(tx.DestinationAccount, "*") {
				f.SourceAccount = tx.SourceAccount
				tx.DestinationAccount = f.DestinationAccount
			}

			// privat really fuck you.
			if strings.HasPrefix(tx.Description, "Переказ на свою картку") &&
				strings.HasPrefix(tx.DestinationAccount, "*") &&
				strings.HasPrefix(f.Description, "Зарахування переказу") &&
				f.SourceAccount == "" {
				tx.DestinationAccount = f.DestinationAccount
				f.SourceAccount = tx.SourceAccount
			}

			// reverse
			if strings.HasPrefix(f.Description, "Переказ на свою картку") &&
				strings.HasPrefix(f.DestinationAccount, "*") &&
				strings.HasPrefix(tx.Description, "Зарахування переказу") &&
				tx.SourceAccount == "" {
				f.DestinationAccount = tx.DestinationAccount
				tx.SourceAccount = f.SourceAccount
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

			f.Type = database.TransactionTypeInternalTransfer
			tx.Type = database.TransactionTypeInternalTransfer

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

func (p *Parser) appendTxOrError(finalTx []*database.Transaction, tx *database.Transaction, err error, raw string, item *Record) []*database.Transaction {
	return appendTxOrError(finalTx, tx, err, raw, item)
}

func appendTxOrError(finalTx []*database.Transaction, tx *database.Transaction, err error, raw string, item *Record) []*database.Transaction {
	if !lo.IsNil(tx) {
		tx.OriginalMessage = item.Message
		finalTx = append(finalTx, tx)
	}

	if !lo.IsNil(err) {
		finalTx = append(finalTx, &database.Transaction{
			Raw:             raw,
			ParsingError:    err,
			OriginalMessage: item.Message,
		})
	}

	return finalTx
}

var (
	simpleExpenseRegex        = regexp.MustCompile(`(\d+.?\d+)([A-Z]{3}) (.*)$`)
	balanceRegex              = regexp.MustCompile(`Бал\. .*(\w{3})`)
	remoteTransferRegex       = simpleExpenseRegex
	incomeTransferRegex       = simpleExpenseRegex
	internalTransferToRegex   = regexp.MustCompile(`(\d+.?\d+)([A-Z]{3}) (Переказ на свою карт[^ ]+ (?:(\d+\*\*\d+) )?(.*))$`)
	internalTransferFromRegex = regexp.MustCompile(`(\d+.?\d+)([A-Z]{3}) (Переказ зі своєї карт[^ ]+ (\*?\d+\*?\*?\d+) ?(.*)?)$`)
)

func (p *Parser) ParseIncomingCardTransfer(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lines := toLines(raw)

	if len(lines) < partialRefundLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", partialRefundLinesCount, len(lines))
	}

	matches := incomeTransferRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:                  uuid.NewString(),
		Date:                date,
		DestinationCurrency: matches[2],
		Description:         matches[3],
		DestinationAmount:   amount,
		Type:                database.TransactionTypeIncome,
		DestinationAccount:  source[0],
		Raw:                 raw,
		DateFromMessage:     source[1],
	}

	return finalTx, nil
}

func (p *Parser) ParsePartialRefund(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lines := toLines(raw)

	if len(lines) < partialRefundLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", incomeTransferLinesCount, len(lines))
	}

	matches := incomeTransferRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:                  uuid.NewString(),
		Date:                date,
		DestinationCurrency: matches[2],
		Description:         matches[3],
		DestinationAmount:   amount,
		Type:                database.TransactionTypeIncome,
		DestinationAccount:  source[0],
		Raw:                 raw,
		DateFromMessage:     source[1],
	}

	return finalTx, nil
}

func (p *Parser) ParseIncomeTransfer(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lines := toLines(raw)

	if len(lines) < incomeTransferLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", incomeTransferLinesCount, len(lines))
	}

	matches := incomeTransferRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:                  uuid.NewString(),
		Date:                date,
		DestinationCurrency: matches[2],
		Description:         matches[3],
		DestinationAmount:   amount,
		Type:                database.TransactionTypeIncome,
		DestinationAccount:  source[0],
		Raw:                 raw,
		DateFromMessage:     source[1],
	}

	return finalTx, nil
}

func (p *Parser) ParseInternalTransfer(
	ctx context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lines := toLines(raw)

	isTo := strings.Contains(strings.ToLower(lines[0]), "переказ на свою карт")

	if isTo {
		return p.parseInternalTransferTo(ctx, raw, lines, date)
	}

	return p.parseInternalTransferFrom(ctx, raw, lines, date)
}

func (p *Parser) parseInternalTransferFrom(
	_ context.Context,
	raw string,
	lines []string,
	date time.Time,
) (*database.Transaction, error) {
	matches := internalTransferFromRegex.FindStringSubmatch(lines[0])

	if len(matches) < 5 {
		return nil, errors.Newf("expected 6 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	destinationAccount := p.formatDestinationAccount(matches[4])

	finalTx := &database.Transaction{
		ID:                          uuid.NewString(),
		Date:                        date,
		DestinationCurrency:         matches[2],
		Description:                 matches[3],
		DestinationAmount:           amount,
		Type:                        database.TransactionTypeInternalTransfer,
		SourceAccount:               destinationAccount,
		DestinationAccount:          source[0],
		InternalTransferDirectionTo: false,
		DateFromMessage:             source[1],
		Raw:                         raw,
	}

	return finalTx, nil
}

func (p *Parser) parseInternalTransferTo(
	_ context.Context,
	raw string,
	lines []string,
	date time.Time,
) (*database.Transaction, error) {
	matches := internalTransferToRegex.FindStringSubmatch(lines[0])

	if len(matches) != 6 && len(matches) != 5 {
		return nil, errors.Newf("expected 5-6 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	destRaw := matches[4]
	if destRaw == "" && len(matches) == 6 {
		destRaw = matches[5]
	}

	destinationAccount := p.formatDestinationAccount(destRaw)

	if !strings.Contains(destinationAccount, "*") {
		destinationAccount = unk
	}

	finalTx := &database.Transaction{
		ID:                          uuid.NewString(),
		Date:                        date,
		SourceCurrency:              matches[2],
		Description:                 matches[3],
		SourceAmount:                amount,
		Type:                        database.TransactionTypeInternalTransfer,
		SourceAccount:               source[0],
		DestinationAccount:          destinationAccount,
		InternalTransferDirectionTo: true,
		DateFromMessage:             source[1],
		Raw:                         raw,
	}

	return finalTx, nil
}

func (p *Parser) formatDestinationAccount(destinationAccount string) string {
	if len(destinationAccount) != 6 {
		return destinationAccount
	}

	return fmt.Sprintf("%s*%s", string(destinationAccount[0]), destinationAccount[4:])
}

func (p *Parser) ParseRemoteTransfer(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lines := toLines(raw)

	if len(lines) < remoteTransferLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", remoteTransferLinesCount, len(lines))
	}

	matches := remoteTransferRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) != 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:              uuid.NewString(),
		Date:            date,
		SourceCurrency:  matches[2],
		Description:     matches[3],
		SourceAmount:    amount,
		Type:            database.TransactionTypeRemoteTransfer,
		SourceAccount:   source[0],
		Raw:             raw,
		DateFromMessage: source[1],
	}

	return finalTx, nil
}

func (p *Parser) ParseCreditPayment(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	lines := strings.Split(raw, "\n")
	if len(lines) < creditPaymentLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", creditPaymentLinesCount, len(lines))
	}

	matches := simpleExpenseRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) < 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:              uuid.NewString(),
		Date:            date,
		SourceCurrency:  matches[2],
		Description:     matches[3],
		SourceAmount:    amount,
		Type:            database.TransactionTypeExpense,
		SourceAccount:   source[0],
		Raw:             raw,
		DateFromMessage: "", // for some reason here its not time.. wtf
	}

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "курс ") { // apply exchange rate logic
			sp := strings.Split(line, " ")

			if len(sp) != 3 {
				return nil, errors.Newf("expected 3 parts for курс, got %v", spew.Sdump(sp))
			}

			currencies := strings.Split(sp[2], "/")
			if len(currencies) != 2 {
				return nil, errors.Newf("expected 2 currencies, got %v", spew.Sdump(currencies))
			}

			rate, rateErr := decimal.NewFromString(sp[1])
			if rateErr != nil {
				return nil, errors.Join(rateErr, errors.Newf("failed to parse rate %s", sp[1]))
			}

			if currencies[1] == finalTx.SourceCurrency {
				finalTx.DestinationCurrency = finalTx.SourceCurrency
				finalTx.DestinationAmount = finalTx.SourceAmount

				finalTx.SourceCurrency = currencies[0]
				finalTx.SourceAmount = amount.Mul(rate)
			} else if currencies[0] == finalTx.SourceCurrency {
				finalTx.DestinationCurrency = finalTx.SourceCurrency
				finalTx.DestinationAmount = finalTx.SourceAmount

				finalTx.SourceCurrency = currencies[1]
				finalTx.SourceAmount = amount.Div(rate)
			} else {
				return nil, errors.Newf("currency mismatch: %s %s", currencies[0], currencies[1])
			}
		}
	}

	for _, line := range lines {
		balMatch := balanceRegex.FindStringSubmatch(line)
		if len(balMatch) != 2 {
			continue
		}

		if balMatch[1] != finalTx.SourceCurrency {
			return nil, errors.Newf("currency mismatch: %s != %s", balMatch[1], finalTx.SourceCurrency)
		}
	}

	return finalTx, nil
}

func (p *Parser) ParseSimpleExpense(
	_ context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	lines := strings.Split(raw, "\n")
	if len(lines) < simpleExpenseLinesCount {
		return nil, errors.Newf("expected %d lines, got %d", simpleExpenseLinesCount, len(lines))
	}

	matches := simpleExpenseRegex.FindStringSubmatch(lines[0])
	if len(matches) != 4 {
		return nil, errors.Newf("expected 4 matches, got %v", spew.Sdump(matches))
	}

	amount, err := decimal.NewFromString(matches[1])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := strings.Split(lines[1], " ")
	if len(source) < 2 {
		return nil, errors.Newf("expected 2 source parts, got %v", spew.Sdump(source))
	}

	finalTx := &database.Transaction{
		ID:              uuid.NewString(),
		Date:            date,
		SourceCurrency:  matches[2],
		Description:     matches[3],
		SourceAmount:    amount,
		Type:            database.TransactionTypeExpense,
		SourceAccount:   source[0],
		Raw:             raw,
		DateFromMessage: source[1],
	}

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "курс ") { // apply exchange rate logic
			sp := strings.Split(line, " ")

			if len(sp) != 3 {
				return nil, errors.Newf("expected 3 parts for курс, got %v", spew.Sdump(sp))
			}

			currencies := strings.Split(sp[2], "/")
			if len(currencies) != 2 {
				return nil, errors.Newf("expected 2 currencies, got %v", spew.Sdump(currencies))
			}

			rate, rateErr := decimal.NewFromString(sp[1])
			if rateErr != nil {
				return nil, errors.Join(rateErr, errors.Newf("failed to parse rate %s", sp[1]))
			}

			if currencies[1] == finalTx.SourceCurrency {
				finalTx.DestinationCurrency = finalTx.SourceCurrency
				finalTx.DestinationAmount = finalTx.SourceAmount

				finalTx.SourceCurrency = currencies[0]
				finalTx.SourceAmount = amount.Mul(rate)
			} else if currencies[0] == finalTx.SourceCurrency {
				finalTx.DestinationCurrency = finalTx.SourceCurrency
				finalTx.DestinationAmount = finalTx.SourceAmount

				finalTx.SourceCurrency = currencies[1]
				finalTx.SourceAmount = amount.Div(rate)
			} else {
				return nil, errors.Newf("currency mismatch: %s %s", currencies[0], currencies[1])
			}
		}
	}

	for _, line := range lines {
		balMatch := balanceRegex.FindStringSubmatch(line)
		if len(balMatch) != 2 {
			continue
		}

		if balMatch[1] != finalTx.SourceCurrency {
			return nil, errors.Newf("currency mismatch: %s != %s", balMatch[1], finalTx.SourceCurrency)
		}
	}

	return finalTx, nil
}
