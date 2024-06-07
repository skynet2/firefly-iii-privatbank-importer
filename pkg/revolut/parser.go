package revolut

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMessages(
	_ context.Context,
	rawArr []*parser.Record,
) ([]*database.Transaction, error) {
	var transactions []*database.Transaction

	for _, raw := range rawArr {
		tx := &database.Transaction{}
		transactions = append(transactions, tx)

		var revolutTx Transaction

		if err := json.Unmarshal(raw.Data, &revolutTx); err != nil {
			tx.ParsingError = errors.Join(err, errors.Newf("invalid json %v", string(raw.Data)))
			continue
		}

		tx.ID = revolutTx.Id
		tx.Date = time.UnixMilli(revolutTx.StartedDate)
		tx.Description = revolutTx.Description

		// todo duplicates
		switch strings.ToUpper(revolutTx.Type) {
		case "TRANSFER":
			if err := p.mapExpense(tx, &revolutTx); err != nil {
				tx.ParsingError = err
				tx.Raw = string(raw.Data)
				continue
			}
		case "EXCHANGE":
			if err := p.mapExchange(tx, &revolutTx); err != nil {
				tx.ParsingError = err
				tx.Raw = string(raw.Data)
			}
		}
	}

	return transactions, nil
}

func (p *Parser) mapExchange(tx *database.Transaction, revolutTx *Transaction) error {
	tx.Type = database.TransactionTypeInternalTransfer // for me this is exchange

	if revolutTx.Amount < 0 {
		tx.SourceCurrency = revolutTx.Currency
		tx.SourceAmount = decimal.NewFromInt(revolutTx.Amount).Div(decimal.NewFromInt(100)).Abs()
		tx.SourceAccount = revolutTx.FromAccount.ID

		tx.DestinationAmount = decimal.NewFromInt(revolutTx.Counterpart.Amount).Div(decimal.NewFromInt(100)).Abs()
		tx.DestinationCurrency = revolutTx.Counterpart.Currency
		tx.DestinationAccount = revolutTx.ToAccount.ID
	} else {
		tx.SourceCurrency = revolutTx.Counterpart.Currency
		tx.SourceAmount = decimal.NewFromInt(revolutTx.Counterpart.Amount).Div(decimal.NewFromInt(100)).Abs()
		tx.SourceAccount = revolutTx.ToAccount.ID

		tx.DestinationAmount = decimal.NewFromInt(revolutTx.Amount).Div(decimal.NewFromInt(100)).Abs()
		tx.DestinationCurrency = revolutTx.Currency
		tx.DestinationAccount = revolutTx.FromAccount.ID
	}

	return nil
}

func (p *Parser) mapExpense(tx *database.Transaction, revolutTx *Transaction) error {
	tx.Type = database.TransactionTypeExpense // for me this is expense

	tx.SourceAmount = decimal.NewFromInt(revolutTx.Amount).Div(decimal.NewFromInt(100)).Abs()
	tx.SourceCurrency = revolutTx.Currency
	tx.SourceAccount = revolutTx.Account.ID

	tx.DestinationAmount = tx.SourceAmount
	tx.DestinationCurrency = revolutTx.Currency

	var notes []string
	notes = append(notes, revolutTx.Tag)

	if revolutTx.CountryCode != "" {
		notes = append(notes, fmt.Sprintf("Country: %v", revolutTx.CountryCode))
	}

	if revolutTx.Recipient.ID != "" {
		notes = append(notes, fmt.Sprintf("Recipient: Type: %v. FirstName: %v. LastName: %v. Code: %v",
			revolutTx.Recipient.Type,
			revolutTx.Recipient.FirstName, revolutTx.Recipient.LastName, revolutTx.Recipient.Code))
	}

	tx.Raw = strings.Join(notes, ", ")
	return nil
}

func (p *Parser) Type() database.TransactionSource {
	return database.Revolut
}
