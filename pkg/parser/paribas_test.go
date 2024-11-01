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

//go:embed testdata/two_expenses.xlsx
var twoExpenses []byte

//go:embed testdata/to_phone.xlsx
var toPhone []byte

//go:embed testdata/blokada_srodkow.xlsx
var blokada []byte

//go:embed testdata/income.xlsx
var income []byte

//go:embed testdata/transfer_to_private_acc.xlsx
var transferToPrivateAccount []byte

//go:embed testdata/credit_card.xlsx
var creditCardPayment []byte

//go:embed testdata/currency_exchange.xlsx
var currencyExchange []byte

//go:embed testdata/transfer_betwee_accounts.xlsx
var betweenAccounts []byte

//go:embed testdata/currency_exchange2.xlsx
var currencyExchange2 []byte

//go:embed testdata/currency_exchange2_2.xlsx
var currencyExchange22 []byte

//go:embed testdata/outgoing_payment_multi_currency.xlsx
var outgoingPaymentMultiCurrency []byte

//go:embed testdata/pshelev_expense.xlsx
var pshelevExpense []byte

//go:embed testdata/account_commission.xlsx
var accountCommission []byte

//go:embed testdata/cash_withdrawal.xlsx
var cashWithdrawal []byte

//go:embed testdata/similar_transfers.xlsx
var similarTransfers []byte

//go:embed testdata/blik_refund.xlsx
var blikRefund []byte

//go:embed testdata/inne_withdrawal.xlsx
var inneWithdrawal []byte

//go:embed testdata/income_multicurrency.xlsx
var incomeMultiCurrency []byte

func TestInneWithdrawal(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: inneWithdrawal,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "8.63", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-04-04 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Name Surname", resp[0].Description)
}

func TestBlikRefund(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: blikRefund,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
	assert.Equal(t, "2699.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "2222222222222222222222", resp[0].DestinationAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-04-05 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Transakcja BLIK, Zwrot dla transakcji TR-U4J-BVPHTYX, Zwrot BLIK internet, Nr 1234566, TERG SPÓŁKA AKCYJNA, REF-12345", resp[0].Description)
}

func TestCashWithdrawal(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: cashWithdrawal,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "600.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "11111111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-03-02 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "1111------222 XX YY zz 4321321 POL 600,00 PLN 2024-03-02", resp[0].Description)
}

func TestSimilarTransfers(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: similarTransfers,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 2)

	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
	assert.Equal(t, "11.68", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "22222222222222222222222222", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-03-01 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Common Desription", resp[0].Description)

	assert.Equal(t, "22.16", resp[1].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[1].SourceCurrency)
}

func TestExpenseCommission(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: accountCommission,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "2.31", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].SourceCurrency)
	assert.Equal(t, "11111111111111111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-24 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Prowizje i opłaty", resp[0].Description)
}

func TestPshelevExpense(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: pshelevExpense,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "200.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "111111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "200.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "2222222222222222222222", resp[0].DestinationAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-08 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "333333", resp[0].Description)
}

func TestToPhone(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: toPhone,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
	assert.Equal(t, "200.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "11111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "200.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "22222222222222222222222", resp[0].DestinationAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-07 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Some Descriptions", resp[0].Description)
}

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

func TestTwoExpenses(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: twoExpenses,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 2)

	assert.NotEqual(t, resp[0].ID, resp[1].ID)
	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "500.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
}

func TestParibasBlokadaSrodkow(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: blokada,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
	assert.Equal(t, "500.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "1234567", resp[0].SourceAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-08 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "PAYPAL  XTB S 111 PL 111______111 500,00 USD ", resp[0].Description)
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

	assert.NoError(t, resp[0].ParsingError)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
	assert.Equal(t, "11.48", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].DestinationCurrency)
	assert.Equal(t, "123443252341234214321331", resp[0].DestinationAccount)

	assert.Equal(t, "11.48", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].SourceCurrency)
	assert.Equal(t, "/es123432523424213132", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-01 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "SOFTWARE DEVELOPMENT SERVICES, INVOICE NO 1-2 XXYY, 31.01.2024", resp[0].Description)
}

func TestParibasIncomeMultiCurrency(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: incomeMultiCurrency,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.NoError(t, resp[0].ParsingError)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
	assert.Equal(t, "102.16", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "EUR", resp[0].DestinationCurrency)
	assert.Equal(t, "22222222222222222222222222", resp[0].DestinationAccount)

	assert.Equal(t, "500.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "11111111111111111111111111", resp[0].SourceAccount)

	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-07-09 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "ID 4444444 5555555555555555555", resp[0].Description)
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

func TestParibasCreditCardRepayment(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: creditCardPayment,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.Equal(t, "PLN", resp[0].SourceCurrency)
	assert.Equal(t, "1.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "1.00", resp[0].DestinationAmount.StringFixed(2))

	assert.Equal(t, "22222222222222222222222222", resp[0].SourceAccount)
	assert.Equal(t, "11111111111111111111111111", resp[0].DestinationAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-10-20 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "Spłata karty", resp[0].Description)
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

func TestParibasCurrencyExchange2(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: currencyExchange2,
		},
		{
			Data: currencyExchange22,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.Equal(t, "EUR", resp[0].SourceCurrency)
	assert.Equal(t, "1390.56", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)
	assert.Equal(t, "6000.00", resp[0].DestinationAmount.StringFixed(2))

	assert.Equal(t, "22222222222222222222222222", resp[0].SourceAccount)
	assert.Equal(t, "11111111111111111111111111", resp[0].DestinationAccount)
	assert.Equal(t, "00:00", resp[0].DateFromMessage)
	assert.Equal(t, "2024-02-08 00:00:00 +0000", resp[0].Date.Format("2006-01-02 15:04:05 -0700"))
	assert.Equal(t, "EUR PLN 4.3148 TWM2402064671", resp[0].Description)
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

func TestSplitExcel(t *testing.T) {
	srv := parser.NewParibas()

	resp, err := srv.SplitExcel(context.TODO(), betweenAccounts)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 2)
}
