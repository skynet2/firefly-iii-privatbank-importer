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

//go:embed testdata/revolut/exchange.csv
var revolutExchange []byte

//go:embed testdata/revolut/exchange_swap.csv
var revolutExchangeSwap []byte

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

func TestRevolutExchange(t *testing.T) {
	srv := parser.NewRevolut()

	resp, err := srv.SplitExcel(context.TODO(), revolutExchange)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	txs, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(resp[0])),
		},
		{
			Data: []byte(hex.EncodeToString(resp[1])),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	assert.EqualValues(t, "2024-10-25 11:48:00", txs[0].Date.Format(time.DateTime))
	assert.EqualValues(t, "EXCHANGE.Exchanged to PLN", txs[0].Description)

	assert.EqualValues(t, "USD", txs[0].SourceCurrency)
	assert.EqualValues(t, "revolut_USD", txs[0].SourceAccount)
	assert.EqualValues(t, "469.57", txs[0].SourceAmount.StringFixed(2))

	assert.EqualValues(t, "PLN", txs[0].DestinationCurrency)
	assert.EqualValues(t, "revolut_PLN", txs[0].DestinationAccount)
	assert.EqualValues(t, "1907.07", txs[0].DestinationAmount.StringFixed(2))
}

func TestRevolutExchangeSwap(t *testing.T) {
	srv := parser.NewRevolut()

	resp, err := srv.SplitExcel(context.TODO(), revolutExchangeSwap)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	txs, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(resp[0])),
		},
		{
			Data: []byte(hex.EncodeToString(resp[1])),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	assert.EqualValues(t, "2024-10-25 11:48:00", txs[0].Date.Format(time.DateTime))
	assert.EqualValues(t, "EXCHANGE.Exchanged to PLN", txs[0].Description)

	assert.EqualValues(t, "USD", txs[0].SourceCurrency)
	assert.EqualValues(t, "revolut_USD", txs[0].SourceAccount)
	assert.EqualValues(t, "469.57", txs[0].SourceAmount.StringFixed(2))

	assert.EqualValues(t, "PLN", txs[0].DestinationCurrency)
	assert.EqualValues(t, "revolut_PLN", txs[0].DestinationAccount)
	assert.EqualValues(t, "1907.07", txs[0].DestinationAmount.StringFixed(2))
}
