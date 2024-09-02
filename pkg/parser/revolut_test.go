package parser_test

import (
	"context"
	_ "embed"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:embed testdata/revolut/simple_expense.csv
var revolutExpense []byte

func TestRevolutSimple(t *testing.T) {
	srv := parser.NewRevolut()

	resp, err := srv.SplitExcel(context.TODO(), revolutExpense)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	txs, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(resp[0])),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	assert.EqualValues(t, "2024-09-02 10:31:35", txs[0].Date.Format(time.DateTime))
	assert.EqualValues(t, "TRANSFER.To XXYYZZ", txs[0].Description)
	assert.EqualValues(t, "USD", txs[0].SourceCurrency)
	assert.EqualValues(t, "21.31", txs[0].SourceAmount.StringFixed(2))
}
