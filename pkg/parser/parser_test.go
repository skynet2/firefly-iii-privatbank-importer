package parser_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

func TestParseSimpleExpense(t *testing.T) {
	input := `1.33USD Розваги. Steam
4*71 16:27
Бал. 1.55USD`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), input, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.33", resp.Amount.String())
	assert.Equal(t, "USD", resp.Currency)
	assert.Equal(t, "Розваги. Steam", resp.Description)
	assert.Equal(t, "4*71", resp.SourceAccount)
	assert.Equal(t, database.TransactionTypeExpense, resp.Type)
}

func TestParseRemoteTransfer(t *testing.T) {
	input := `1.00UAH Переказ через Приват24 Одержувач: Імя Фамілія ПоБатькові
4*68 16:41
Бал. 17.81UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), input, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp.Amount.StringFixed(2))
	assert.Equal(t, "UAH", resp.Currency)
	assert.Equal(t, "Переказ через Приват24 Одержувач: Імя Фамілія ПоБатькові", resp.Description)
	assert.Equal(t, "4*68", resp.SourceAccount)
	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp.Type)
}

func TestParseInternalTransferTo(t *testing.T) {
	input := `1.00UAH Переказ на свою карту 51**20 через додаток Приват24
4*68 16:13
Бал. 18.81UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), input, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp.Amount.StringFixed(2))
	assert.Equal(t, "UAH", resp.Currency)
	assert.Equal(t, "Переказ на свою карту 51**20 через додаток Приват24", resp.Description)
	assert.Equal(t, "4*68", resp.SourceAccount)
	assert.Equal(t, "5*20", resp.DestinationAccount)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp.Type)
}

func TestParseInternalTransferFrom(t *testing.T) {
	input := `1.00UAH Переказ зі своєї карти 47**68 через додаток Приват24
5*20 16:13
Бал. 123.32UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), input, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp.Amount.StringFixed(2))
	assert.Equal(t, "UAH", resp.Currency)
	assert.Equal(t, "Переказ зі своєї карти 47**68 через додаток Приват24", resp.Description)
	assert.Equal(t, "5*20", resp.SourceAccount)
	assert.Equal(t, "4*68", resp.DestinationAccount)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp.Type)
}

func TestParseIncomeTransfer(t *testing.T) {
	input := `123.11UAH Переказ через Приват24 Відправник: Імя Фамілія ПоБатькові
5*20 20:11
Бал. 11111.22UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), input, time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "123.11", resp.Amount.StringFixed(2))
	assert.Equal(t, "UAH", resp.Currency)
	assert.Equal(t, "Переказ через Приват24 Відправник: Імя Фамілія ПоБатькові", resp.Description)
	assert.Equal(t, "5*20", resp.DestinationAccount)
	assert.Equal(t, database.TransactionTypeIncome, resp.Type)
}
