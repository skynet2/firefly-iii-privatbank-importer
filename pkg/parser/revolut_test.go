package parser_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:embed testdata/revolut/card_expense_multi_currency.json
var revolutCardExpenseMultiCurrency []byte

//go:embed testdata/revolut/deposit_by_card.json
var revolutDepositByCard []byte

func TestCardExpenseMultiCurrency(t *testing.T) {
	rev := parser.NewRevolut()

	transactions, err := rev.ParseMessages(context.Background(), []*parser.Record{
		{
			Data: revolutCardExpenseMultiCurrency,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)

	assert.EqualValues(t, "1024fa59-9b59-409b-bb52-211b05730ca1", transactions[0].ID)
	assert.EqualValues(t, "2024-04-08 16:58:55 +0200 CEST", transactions[0].DateFromMessage)

	assert.EqualValues(t, database.TransactionTypeExpense, transactions[0].Type)
	assert.EqualValues(t, "USD", transactions[0].SourceCurrency)
	assert.EqualValues(t, "29.46", transactions[0].SourceAmount.StringFixed(2))
	assert.EqualValues(t, "source-account-id", transactions[0].SourceAccount)

	assert.EqualValues(t, "PLN", transactions[0].DestinationCurrency)
	assert.EqualValues(t, "115.60", transactions[0].DestinationAmount.StringFixed(2))
	assert.EqualValues(t, "pyszne.pl", transactions[0].Description)
}

func TestCardDeposit(t *testing.T) {
	rev := parser.NewRevolut()

	transactions, err := rev.ParseMessages(context.Background(), []*parser.Record{
		{
			Data: revolutDepositByCard,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)

	assert.EqualValues(t, "14e1d940-50e3-4579-ae86-b5280df28909", transactions[0].ID)
	assert.EqualValues(t, "2024-04-07 13:39:08 +0200 CEST", transactions[0].DateFromMessage)

	assert.EqualValues(t, database.TransactionTypeIncome, transactions[0].Type)
	assert.EqualValues(t, "USD", transactions[0].SourceCurrency)
	assert.EqualValues(t, "USD", transactions[0].DestinationCurrency)
	assert.EqualValues(t, "Monodirectfc", transactions[0].Description)
	assert.EqualValues(t, "750.00", transactions[0].DestinationAmount.StringFixed(2))

	assert.EqualValues(t, "receiver-account-id", transactions[0].DestinationAccount)
}
