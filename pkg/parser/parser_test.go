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

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "1.33", resp[0].SourceAmount.String())
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "Розваги. Steam", resp[0].Description)
	assert.Equal(t, "4*71", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
}

func TestParseSimpleExpense2(t *testing.T) {
	input := `83.69PLN Інтернет-магазини. AliExpress
4*67 12:19
Бал. 12.82USD
Курс 0.2558 USD/PLN`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "83.69", resp[0].DestinationAmount.String())
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)

	assert.Equal(t, "21.41", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "Інтернет-магазини. AliExpress", resp[0].Description)
	assert.Equal(t, "4*67", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)
}

func TestParseRemoteTransfer(t *testing.T) {
	input := `1.00UAH Переказ через Приват24 Одержувач: Імя Фамілія ПоБатькові
4*68 16:41
Бал. 17.81UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].SourceCurrency)
	assert.Equal(t, "Переказ через Приват24 Одержувач: Імя Фамілія ПоБатькові", resp[0].Description)
	assert.Equal(t, "4*68", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
}

func TestParseRemoteTransfer3(t *testing.T) {
	input := `715.06UAH Переказ зі своєї карти
5*20 19:29
Бал. 111.29UAH
Кред. лiмiт 111.0UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "715.06", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].SourceCurrency)
	assert.Equal(t, "Переказ зі своєї карти", resp[0].Description)
	assert.Equal(t, "5*20", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
}

func TestParseRemoteTransfer2(t *testing.T) {
	input := `4000.00UAH Переказ через додаток Приват24. Одержувач: ХХ УУ ММ
5*20 12:58
Бал. 123.32UAH
Кред. лiмiт 3000000.0UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "4000.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].SourceCurrency)
	assert.Equal(t, "Переказ через додаток Приват24. Одержувач: ХХ УУ ММ", resp[0].Description)
	assert.Equal(t, "5*20", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeRemoteTransfer, resp[0].Type)
}

func TestParseInternalTransferTo(t *testing.T) {
	input := `1.00UAH Переказ на свою карту 51**20 через додаток Приват24
4*68 16:13
Бал. 18.81UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].SourceCurrency)
	assert.Equal(t, "Переказ на свою карту 51**20 через додаток Приват24", resp[0].Description)
	assert.Equal(t, "4*68", resp[0].SourceAccount)
	assert.Equal(t, "5*20", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.True(t, resp[0].InternalTransferDirectionTo)
}

func TestParseInternalTransferFrom(t *testing.T) {
	input := `1.00UAH Переказ зі своєї карти 47**68 через додаток Приват24
5*20 16:13
Бал. 123.32UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "0.00", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "", resp[0].SourceCurrency)

	assert.Equal(t, "1.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)

	assert.Equal(t, "Переказ зі своєї карти 47**68 через додаток Приват24", resp[0].Description)
	assert.Equal(t, "4*68", resp[0].SourceAccount)
	assert.Equal(t, "5*20", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.False(t, resp[0].InternalTransferDirectionTo)
}

func TestParseInternalTransferFromUSD(t *testing.T) {
	input := `1.00USD Переказ зі своєї карти 52**20 через додаток Приват24
4*71 23:50
Бал. 89.19USD`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "1.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "Переказ зі своєї карти 52**20 через додаток Приват24", resp[0].Description)
	assert.Equal(t, "5*20", resp[0].SourceAccount)
	assert.Equal(t, "4*71", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	assert.False(t, resp[0].InternalTransferDirectionTo)
}

func TestParseIncomeTransfer(t *testing.T) {
	input := `123.11UAH Переказ через Приват24 Відправник: Імя Фамілія ПоБатькові
5*20 20:11
Бал. 11111.22UAH`

	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(input),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "123.11", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)
	assert.Equal(t, "Переказ через Приват24 Відправник: Імя Фамілія ПоБатькові", resp[0].Description)
	assert.Equal(t, "5*20", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
}
