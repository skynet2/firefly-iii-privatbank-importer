package parser_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:embed testdata/zen/income.csv
var zenIncome []byte

func TestParseIncome(t *testing.T) {
	srv := parser.NewZen()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: zenIncome,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 2)

	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
	assert.Equal(t, "7.24", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].DestinationCurrency)
	assert.Equal(t, "zen_EUR", resp[0].DestinationAccount)

	assert.Equal(t, "2023-04-27 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "eCommerce settlement: <some_id>", resp[0].Description)

	assert.Equal(t, database.TransactionTypeExpense, resp[1].Type)
	assert.Equal(t, "0.70", resp[1].SourceAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[1].SourceCurrency)
	assert.Equal(t, "zen_EUR", resp[1].SourceAccount)
	assert.Contains(t, resp[1].Description, "settlement diff for")
}
