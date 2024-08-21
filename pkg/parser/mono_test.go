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

//go:embed testdata/mono/chargeoff.csv
var monoChargeOff []byte

func TestParseMonoSimpleExpense(t *testing.T) {
	mono := parser.NewMono()

	resp, err := mono.SplitExcel(context.TODO(), monoChargeOff)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)

	txs, err := mono.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(resp[0])),
		},
	})
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	assert.EqualValues(t, "2024-08-11 12:19:14", txs[0].Date.Format(time.DateTime))
	assert.EqualValues(t, "Списання Allegro", txs[0].Description)
	assert.EqualValues(t, "UAH", txs[0].SourceCurrency)
	assert.EqualValues(t, "1231.79", txs[0].SourceAmount.StringFixed(2))

	assert.EqualValues(t, "PLN", txs[0].DestinationCurrency)
	assert.EqualValues(t, "128.71", txs[0].DestinationAmount.StringFixed(2))
}
