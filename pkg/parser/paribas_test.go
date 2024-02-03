package parser_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:embed testdata/blik.xlsx
var blik []byte

//go:embed testdata/income.xlsx
var income []byte

//go:embed testdata/transfer_to_private_acc.xlsx
var transferToPrivateAccount []byte

//go:embed testdata/currency_exchange.xlsx
var currencyExchange []byte

//go:embed testdata/transfer_betwee_accounts.xlsx
var betweenAccounts []byte

//go:embed testdata/outgoing_payment_multi_currency.xlsx
var outgoingPaymentMultiCurrency []byte

func TestParibasBlik(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: blik,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "119.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "11112222333344455556777", resp[0].SourceAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-02 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Transakcja BLIK, Allegro xxxx-c21, Płatność BLIK w internecie, Nr 12324, ALLEGRO SP. Z O.O., allegro.pl", resp[0].Description)
}

func TestMultiCurrencyPayment(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: outgoingPaymentMultiCurrency,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)

	assert.Equal(t, "837.89", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "22222222222222222222222", resp[0].SourceAccount)

	assert.Equal(t, "199.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].DestinationCurrency)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-01-08 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "xx-----yy 4,xx xx.yyy my.vmware.com DRI-VMware IRL 199,00 USD 2024-01-08", resp[0].Description)
}

func TestParibasIncome(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: income,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
	assert.Equal(t, "11.48", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].DestinationCurrency)
	assert.Equal(t, "123443252341234214321331", resp[0].DestinationAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-01 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "SOFTWARE DEVELOPMENT SERVICES, INVOICE NO 1-2 XXYY, 31.01.2024", resp[0].Description)
}

func TestParibasTransferToPrivateAccount(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: transferToPrivateAccount,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "1200.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "1200.00", resp[0].DestinationAmount.StringFixed(2))

	assert.Equal(t, "22222222222222222222222222222222", resp[0].SourceAccount)
	assert.Equal(t, "1111111111111111111111111111", resp[0].DestinationAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-01 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Przelew środków", resp[0].Description)
}

func TestParibasCurrencyExchange(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: currencyExchange,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "1500.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "6000.90", resp[0].DestinationAmount.StringFixed(2))

	assert.Equal(t, "1111111111111", resp[0].SourceAccount)
	assert.Equal(t, "22222222222222222", resp[0].DestinationAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-01-24 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "USD PLN 4.0006 TWM2131232132131", resp[0].Description)
}

func TestParibasBetweenAccounts(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: betweenAccounts,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)
	assert.Len(t, resp[0].DuplicateTransactions, 1)

	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].DuplicateTransactions[0].Type)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "1200.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "1200.00", resp[0].DestinationAmount.StringFixed(2))

	assert.Equal(t, "11111111111111111111111111", resp[0].DestinationAccount)
	assert.Equal(t, "22222222222222222222222222", resp[0].SourceAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-01 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Przelew środków", resp[0].Description)
}
