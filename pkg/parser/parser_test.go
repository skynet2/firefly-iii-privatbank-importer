package parser_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

func TestExpensesShouldNotMerged(t *testing.T) {
	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(`89.80PLN Ресторани, кафе, бари. Pyszne.pl, Wroclaw
4*67 16:17
Бал. 1.86USD
Курс 0.2547 USD/PLN`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
		{
			Data: []byte(`8.98PLN Ресторани, кафе, бари. Pyszne.pl, Wroclaw
4*67 16:18
Бал. 236.57USD
Курс 0.2550 USD/PLN`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 2)

	assert.Equal(t, "22.87", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "4*67", resp[0].SourceAccount)

	assert.Equal(t, "89.80", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "", resp[0].DestinationAccount)
	assert.Equal(t, "PLN", resp[0].DestinationCurrency)

	assert.Equal(t, "Ресторани, кафе, бари. Pyszne.pl, Wroclaw", resp[0].Description)
	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)

	assert.Equal(t, "2.29", resp[1].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[1].SourceCurrency)
	assert.Equal(t, "4*67", resp[1].SourceAccount)

	assert.Equal(t, "8.98", resp[1].DestinationAmount.StringFixed(2))
	assert.Equal(t, "", resp[1].DestinationAccount)
	assert.Equal(t, "PLN", resp[1].DestinationCurrency)

	assert.Equal(t, "Ресторани, кафе, бари. Pyszne.pl, Wroclaw", resp[1].Description)
	assert.Equal(t, database.TransactionTypeExpense, resp[1].Type)
}

func TestTransferBetweenOwnCards(t *testing.T) {
	srv := parser.NewParser()

	t.Run("order 1", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`1100.00USD Переказ зі своєї карти
4*59 22:40
Бал. 1.21USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`1100.00USD Зарахування переказу на картку
4*71 22:40
Бал. 1.60USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "1100.00", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "1100.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "4*71", resp[0].DestinationAccount)
		assert.Equal(t, "USD", resp[0].DestinationCurrency)

		assert.Equal(t, "Переказ зі своєї карти", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
	t.Run("order 2", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`1100.00USD Зарахування переказу на картку
4*71 22:40
Бал. 1.60USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`1100.00USD Переказ зі своєї карти
4*59 22:40
Бал. 1.21USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "1100.00", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "1100.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "4*71", resp[0].DestinationAccount)
		assert.Equal(t, "USD", resp[0].DestinationCurrency)

		assert.Equal(t, "Зарахування переказу на картку", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
}

func TestTransferBetweenOwnCards3(t *testing.T) {
	srv := parser.NewParser()

	t.Run("order 1", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`40.00USD Зарахування переказу. *1959
4*71 11:11
Бал. 1.37USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`40.00USD Переказ на свою картку. *6471
4*59 11:11
Бал. 1446.74USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "40.00", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "40.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "4*71", resp[0].DestinationAccount)
		assert.Equal(t, "USD", resp[0].DestinationCurrency)

		assert.Equal(t, "Зарахування переказу. *1959", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
	t.Run("order 2", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`40.00USD Переказ на свою картку. *6471
4*59 11:11
Бал. 1446.74USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`40.00USD Зарахування переказу. *1959
4*71 11:11
Бал. 1.37USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "40.00", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "40.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "4*71", resp[0].DestinationAccount)
		assert.Equal(t, "USD", resp[0].DestinationCurrency)

		assert.Equal(t, "Переказ на свою картку. *6471", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
}

func TestTransferBetweenOwnCards2(t *testing.T) {
	srv := parser.NewParser()

	t.Run("order 1", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`356.45USD Переказ на свою картку. *0320
4*59 05:06
Бал. 123.96USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`14418.00UAH Переказ зі своєї картки *1959
5*20 05:06
Бал. 123.00UAH`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "356.45", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "14418.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "5*20", resp[0].DestinationAccount)
		assert.Equal(t, "UAH", resp[0].DestinationCurrency)

		assert.Equal(t, "Переказ на свою картку. *0320", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
	t.Run("order 2", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`14418.00UAH Переказ зі своєї картки *1959
5*20 05:06
Бал. 123.00UAH`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`356.45USD Переказ на свою картку. *0320
4*59 05:06
Бал. 123.96USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "356.45", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "14418.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "5*20", resp[0].DestinationAccount)
		assert.Equal(t, "UAH", resp[0].DestinationCurrency)

		assert.Equal(t, "Переказ зі своєї картки *1959", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
}

func TestConvertCurrency3(t *testing.T) {
	srv := parser.NewParser()

	t.Run("opt1", func(t *testing.T) {
		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(`5200.00UAH Переказ зі своєї карти 46**59 через додаток Приват24
5*20 18:51
Бал. 1.84UAH`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(`133.78USD Переказ на свою картку через додаток Приват24
4*59 18:51
Бал. 1.95USD`),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "133.78", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)
		assert.Equal(t, "4*59", resp[0].SourceAccount)

		assert.Equal(t, "5200.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "5*20", resp[0].DestinationAccount)
		assert.Equal(t, "UAH", resp[0].DestinationCurrency)

		assert.Equal(t, "Переказ зі своєї карти 46**59 через додаток Приват24", resp[0].Description)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
}

func TestParseCurrencyExchange4(t *testing.T) {
	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(`1723.64USD Переказ на свою карту через Приват24
4*67 15:16
Бал. 123.59USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
		{
			Data: []byte(`65550.00UAH Переказ зі своєї картки 46**67 через Приват24
5*20 15:16
Бал. 123.59UAH
Кред. лiмiт 300000.0UAH`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "1723.64", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "4*67", resp[0].SourceAccount)

	assert.Equal(t, "65550.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "5*20", resp[0].DestinationAccount)
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)

	assert.Equal(t, "Переказ на свою карту через Приват24", resp[0].Description)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
}

func TestParseCurrencyExchange3(t *testing.T) {
	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(`65550.00UAH Переказ зі своєї картки 46**67 через Приват24
5*20 15:16
Бал. 123.59UAH
Кред. лiмiт 300000.0UAH`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
		{
			Data: []byte(`1723.64USD Переказ на свою карту через Приват24
4*67 15:16
Бал. 123.59USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "1723.64", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "4*67", resp[0].SourceAccount)

	assert.Equal(t, "65550.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "5*20", resp[0].DestinationAccount)
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)

	assert.Equal(t, "Переказ зі своєї картки 46**67 через Приват24", resp[0].Description)
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
}

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

func TestParseSimpleRefund(t *testing.T) {
	input := `120.52UAH Повернення. Транспорт. xx.yy
4*68 15:09
Бал. 329.89UAH`

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

	assert.Equal(t, "120.52", resp[0].DestinationAmount.String())
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)
	assert.Equal(t, "Повернення. Транспорт. xx.yy", resp[0].Description)
	assert.Equal(t, "4*68", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
}

func TestNewTransfer(t *testing.T) {
	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
		{
			Data: []byte(`40.00USD Переказ на свою карту через Приват24
4*59 22:28
Бал. 1486.74USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
		{
			Data: []byte(`40.00USD Зарахування переказу через Приват24 зі своєї картки
4*71 22:28
Бал. 61.20USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "40", resp[0].DestinationAmount.String())
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "Переказ на свою карту через Приват24", resp[0].Description)
	assert.Equal(t, "4*71", resp[0].DestinationAccount)

	assert.Equal(t, "4*59", resp[0].SourceAccount)
	assert.Equal(t, "40", resp[0].SourceAmount.String())
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
}

func TestNewTransfer2(t *testing.T) {
	srv := parser.NewParser()

	resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{

		{
			Data: []byte(`40.00USD Зарахування переказу через Приват24 зі своєї картки
4*71 22:28
Бал. 61.20USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
		{
			Data: []byte(`40.00USD Переказ на свою карту через Приват24
4*59 22:28
Бал. 1486.74USD`),
			Message: &database.Message{
				CreatedAt: time.Now(),
			},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)

	assert.Equal(t, "40", resp[0].DestinationAmount.String())
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "Зарахування переказу через Приват24 зі своєї картки", resp[0].Description)
	assert.Equal(t, "4*71", resp[0].DestinationAccount)

	assert.Equal(t, "4*59", resp[0].SourceAccount)
	assert.Equal(t, "40", resp[0].SourceAmount.String())
	assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
}

func TestIncomeTransferP2P(t *testing.T) {
	input := `1466.60USD Зарахування переказу з картки через Приват24
4*51 22:00
Бал. 1.89USD`

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

	assert.Equal(t, "1466.60", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "Зарахування переказу з картки через Приват24", resp[0].Description)
	assert.Equal(t, "4*51", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)

	_, err = srv.Merge(context.TODO(), resp)
	assert.NoError(t, err)
}

func TestCreditExpense(t *testing.T) {
	input := `1466.60UAH Списання
4*40 20.05.24`

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

	assert.Equal(t, "1466.60", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].SourceCurrency)
	assert.Equal(t, "Списання", resp[0].Description)
	assert.Equal(t, "4*40", resp[0].SourceAccount)
	assert.Equal(t, database.TransactionTypeExpense, resp[0].Type)

	_, err = srv.Merge(context.TODO(), resp)
	assert.NoError(t, err)
}

func TestPartialRefund(t *testing.T) {
	input := `1.01USD Зарахування
4*71 09.03.24`

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

	assert.Equal(t, "1.01", resp[0].DestinationAmount.String())
	assert.Equal(t, "USD", resp[0].DestinationCurrency)
	assert.Equal(t, "Зарахування", resp[0].Description)
	assert.Equal(t, "4*71", resp[0].DestinationAccount)
	assert.Equal(t, database.TransactionTypeIncome, resp[0].Type)
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

func TestMultiCurrencyExpense(t *testing.T) {
	input := `249.00UAH Комуналка та Інтернет. Sweet TV
4*71 22:19
Бал. 45.45USD
Курс 37.3873 UAH/USD`

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

	assert.Equal(t, "249.00", resp[0].DestinationAmount.StringFixed(2))
	assert.Equal(t, "UAH", resp[0].DestinationCurrency)

	assert.Equal(t, "6.66", resp[0].SourceAmount.StringFixed(2))
	assert.Equal(t, "USD", resp[0].SourceCurrency)
	assert.Equal(t, "Комуналка та Інтернет. Sweet TV", resp[0].Description)
	assert.Equal(t, "4*71", resp[0].SourceAccount)
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

func TestMerger(t *testing.T) {
	t.Run("firstIsTo", func(t *testing.T) {
		pr := parser.NewParser()

		txList := []*database.Transaction{
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: true,
			},
			{
				ID:             uuid.NewString(),
				Type:           database.TransactionTypeExpense,
				SourceCurrency: "USD",
				SourceAccount:  "4*71",
				SourceAmount:   decimal.RequireFromString("1.33"),
			},
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: false,
			},
		}
		resp, err := pr.Merge(context.TODO(), txList)

		assert.NoError(t, err)
		assert.Len(t, resp, 2)

		assert.Equal(t, txList[0].ID, resp[0].ID)
		assert.Len(t, resp[0].DuplicateTransactions, 1)
		assert.Equal(t, txList[2].ID, resp[0].DuplicateTransactions[0].ID)

		assert.Equal(t, txList[1].ID, resp[1].ID)
	})

	t.Run("firstIsFrom", func(t *testing.T) {
		pr := parser.NewParser()

		txList := []*database.Transaction{
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: false,
			},
			{
				ID:             uuid.NewString(),
				Type:           database.TransactionTypeExpense,
				SourceCurrency: "USD",
				SourceAccount:  "4*71",
				SourceAmount:   decimal.RequireFromString("1.33"),
			},
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: true,
			},
		}
		resp, err := pr.Merge(context.TODO(), txList)

		assert.NoError(t, err)
		assert.Len(t, resp, 2)

		assert.Equal(t, txList[0].ID, resp[0].ID)
		assert.Len(t, resp[0].DuplicateTransactions, 1)
		assert.Equal(t, txList[2].ID, resp[0].DuplicateTransactions[0].ID)

		assert.Equal(t, txList[1].ID, resp[1].ID)
	})

	t.Run("multi currency", func(t *testing.T) {
		pr := parser.NewParser()

		txList := []*database.Transaction{
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				DestinationCurrency:         "USD",
				SourceAccount:               "4*68",
				DestinationAmount:           decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: false,
			},
			{
				ID:             uuid.NewString(),
				Type:           database.TransactionTypeExpense,
				SourceCurrency: "USD",
				SourceAccount:  "4*71",
				SourceAmount:   decimal.RequireFromString("1.33"),
			},
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("38.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: true,
			},
		}
		resp, err := pr.Merge(context.TODO(), txList)

		assert.NoError(t, err)
		assert.Len(t, resp, 2)

		assert.Equal(t, txList[0].ID, resp[0].ID)
		assert.Len(t, resp[0].DuplicateTransactions, 1)
		assert.Equal(t, txList[2].ID, resp[0].DuplicateTransactions[0].ID)

		assert.Equal(t, txList[1].ID, resp[1].ID)

		assert.Equal(t, "USD", resp[0].DestinationCurrency)
		assert.Equal(t, "UAH", resp[0].SourceCurrency)

		assert.Equal(t, "1.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "38.00", resp[0].SourceAmount.StringFixed(2))
	})

	t.Run("multi currency 2", func(t *testing.T) {
		pr := parser.NewParser()

		txList := []*database.Transaction{
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				SourceCurrency:              "UAH",
				SourceAccount:               "4*68",
				SourceAmount:                decimal.RequireFromString("38.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: true,
			},
			{
				ID:                          uuid.NewString(),
				Type:                        database.TransactionTypeInternalTransfer,
				DestinationCurrency:         "USD",
				SourceAccount:               "4*68",
				DestinationAmount:           decimal.RequireFromString("1.00"),
				DestinationAccount:          "5*20",
				DateFromMessage:             "16:13",
				InternalTransferDirectionTo: false,
			},
			{
				ID:             uuid.NewString(),
				Type:           database.TransactionTypeExpense,
				SourceCurrency: "USD",
				SourceAccount:  "4*71",
				SourceAmount:   decimal.RequireFromString("1.33"),
			},
		}
		resp, err := pr.Merge(context.TODO(), txList)

		assert.NoError(t, err)
		assert.Len(t, resp, 2)

		assert.Equal(t, txList[0].ID, resp[0].ID)
		assert.Len(t, resp[0].DuplicateTransactions, 1)
		assert.Equal(t, txList[1].ID, resp[0].DuplicateTransactions[0].ID)

		assert.Equal(t, txList[2].ID, resp[1].ID)

		assert.Equal(t, "USD", resp[0].DestinationCurrency)
		assert.Equal(t, "UAH", resp[0].SourceCurrency)

		assert.Equal(t, "1.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "38.00", resp[0].SourceAmount.StringFixed(2))
	})

	t.Run("test currency exchange", func(t *testing.T) {
		input1 := `123.54EUR Переказ на свою карту 52**20 через Приват24
5*60 22:02
Бал. 0.00EUR`

		input2 := `5555.99UAH Переказ зі своєї картки 52**60 через Приват24
5*20 22:02
Бал. 123.28UAH
Кред. лiмiт 300000.0UAH`

		srv := parser.NewParser()

		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(input1),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(input2),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "5555.99", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "UAH", resp[0].DestinationCurrency)

		assert.Equal(t, "123.54", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "EUR", resp[0].SourceCurrency)

		assert.Equal(t, "Переказ на свою карту 52**20 через Приват24", resp[0].Description)
		assert.Equal(t, "5*60", resp[0].SourceAccount)
		assert.Equal(t, "5*20", resp[0].DestinationAccount)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})

	t.Run("test transfer between account same currency", func(t *testing.T) {
		input1 := `100.00USD Переказ на свою карту 47**71 через Приват24
4*67 11:19
Бал. 123.00USD`

		input2 := `100.00USD Переказ зі своєї картки 46**67 через Приват24
4*71 11:20
Бал. 178.65USD`

		srv := parser.NewParser()

		resp, err := srv.ParseMessages(context.TODO(), []*parser.Record{
			{
				Data: []byte(input1),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
			{
				Data: []byte(input2),
				Message: &database.Message{
					CreatedAt: time.Now(),
				},
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp, 1)

		assert.Equal(t, "100.00", resp[0].DestinationAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].DestinationCurrency)

		assert.Equal(t, "100.00", resp[0].SourceAmount.StringFixed(2))
		assert.Equal(t, "USD", resp[0].SourceCurrency)

		assert.Equal(t, "Переказ на свою карту 47**71 через Приват24", resp[0].Description)
		assert.Equal(t, "4*67", resp[0].SourceAccount)
		assert.Equal(t, "4*71", resp[0].DestinationAccount)
		assert.Equal(t, database.TransactionTypeInternalTransfer, resp[0].Type)
	})
}
