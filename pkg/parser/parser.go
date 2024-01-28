package parser

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	simpleExpenseLinesCount  = 3
	remoteTransferLinesCount = 3
)

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMessages(
	ctx context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	lower := strings.ToLower(raw)

	if strings.Contains(lower, "переказ на свою карту") || strings.Contains(lower, "переказ зі своєї карти") { // internal transfer
		return p.ParseInternalTransfer(ctx, raw, date)
	}

	if strings.Contains(lower, "переказ через приват") { // remote transfer
		return p.ParseRemoteTransfer(ctx, raw, date)
	}

	return p.ParseSimpleExpense(ctx, raw, date)
}

var (
	simpleExpenseRegex        = regexp.MustCompile("(\\d+.?\\d+)([A-Z]{3}) (.*)$")
	remoteTransferRegex       = simpleExpenseRegex
	internalTransferToRegex   = regexp.MustCompile(`(\d+.?\d+)([A-Z]{3}) (Переказ на свою карту (\d+\*\*\d+) (.*))$`)
	internalTransferFromRegex = regexp.MustCompile(`(\d+.?\d+)([A-Z]{3}) (Переказ зі своєї карти (\d+\*\*\d+) (.*))$`)
)

func (p *Parser) ParseInternalTransfer(
	ctx context.Context,
	raw string,
	date time.Time,
) (*database.Transaction, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")

	isTo := strings.Contains(strings.ToLower(lines[0]), "переказ на свою карту")

	if isTo {
		return p.parseInternalTransferTo(ctx, lines, date)
	}

	return p.parseInternalTransferFrom(ctx, lines, date)
}

func (p *Parser) parseInternalTransferFrom(
	_ context.Context,
	lines []string,
	date time.Time,
) (*database.Transaction, error) {
	matches := internalTransferFromRegex.FindStringSubmatch(lines[0])

	if len(matches) != 6 {
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
		Currency:                    matches[2],
		Description:                 matches[3],
		Amount:                      amount,
		Type:                        database.TransactionTypeInternalTransfer,
		SourceAccount:               source[0],
		DestinationAccount:          destinationAccount,
		InternalTransferDirectionTo: false,
		DateFromMessage:             source[1],
	}

	return finalTx, nil
}

func (p *Parser) parseInternalTransferTo(
	_ context.Context,
	lines []string,
	date time.Time,
) (*database.Transaction, error) {
	matches := internalTransferToRegex.FindStringSubmatch(lines[0])

	if len(matches) != 6 {
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
		Currency:                    matches[2],
		Description:                 matches[3],
		Amount:                      amount,
		Type:                        database.TransactionTypeInternalTransfer,
		SourceAccount:               source[0],
		DestinationAccount:          destinationAccount,
		InternalTransferDirectionTo: true,
		DateFromMessage:             source[1],
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
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
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
		ID:            uuid.NewString(),
		Date:          date,
		Currency:      matches[2],
		Description:   matches[3],
		Amount:        amount,
		Type:          database.TransactionTypeRemoteTransfer,
		SourceAccount: source[0],
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
	if len(lines) != simpleExpenseLinesCount {
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
		ID:            uuid.NewString(),
		Date:          date,
		Currency:      matches[2],
		Description:   matches[3],
		Amount:        amount,
		Type:          database.TransactionTypeExpense,
		SourceAccount: source[0],
	}

	return finalTx, nil
}
