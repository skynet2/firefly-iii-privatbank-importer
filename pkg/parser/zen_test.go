package parser_test

import (
	"context"
	_ "embed"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:embed testdata/zen/income.csv
var zenIncome []byte

//go:embed testdata/zen/split.csv
var zenSplit []byte

func TestSplit(t *testing.T) {
	srv := parser.NewZen()

	resp, err := srv.SplitExcel(context.TODO(), zenSplit)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Len(t, resp, 7)
	assert.EqualValues(t, "1-Jun-24,E-commerce recon,eCommerce settlement: 1234,5.69,USD,5,USD,1,Fee charge in the name of ZEN Technology B.V. for technical processing,-1.31,USD,\n",
		string(resp[0]))

	assert.EqualValues(t, "19-Jun-24,Card payment,PAYPAL *user              LUX CARD: MASTERCARD *1122,-193.57,USD,-193.57,USD,1,Fee for processing transaction,,,\n",
		string(resp[6]))

	resp2, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(resp[0])),
		},
		{
			Data: []byte(hex.EncodeToString(resp[1])),
		},
	})

	assert.NoError(t, err)
	assert.Len(t, resp2, 3)

	assert.NoError(t, resp2[0].ParsingError)
	assert.NoError(t, resp2[1].ParsingError)
}

func TestParseIncome(t *testing.T) {
	srv := parser.NewZen()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(hex.EncodeToString(zenIncome)),
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

//func TestSplitBigFile(t *testing.T) {
//	ff, err := os.OpenFile("/mnt/c/Users/iqpir/Downloads/statement-USD-06_2024.csv", os.O_RDONLY, 0)
//	assert.NoError(t, err)
//
//	data, err := io.ReadAll(ff)
//	assert.NoError(t, err)
//	srv := parser.NewZen()
//
//	resp, err := srv.SplitExcel(context.TODO(), data)
//	assert.NoError(t, err)
//	assert.NotNil(t, resp)
//}
