package processor_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

func TestMerger(t *testing.T) {
	t.Run("firstIsTo", func(t *testing.T) {
		pr := processor.NewProcessor(nil, nil, nil, nil)

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
		pr := processor.NewProcessor(nil, nil, nil, nil)

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
		pr := processor.NewProcessor(nil, nil, nil, nil)

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
		pr := processor.NewProcessor(nil, nil, nil, nil)

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
}
