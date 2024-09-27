package printer_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/common"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/printer"
)

func TestPrinter_Commit(t *testing.T) {
	p := printer.NewPrinter()
	ctx := context.Background()

	mappedTx := []*firefly.MappedTransaction{
		{
			Original: &database.Transaction{
				TransactionSource: "Bank",
				Date:              time.Now(),
			},
		},
	}

	errArr := []error{}
	result := p.Commit(ctx, mappedTx, errArr)

	assert.NotEmpty(t, result, "Result should not be empty")
}

func TestPrinter_Dry(t *testing.T) {
	p := printer.NewPrinter()
	ctx := context.Background()

	mappedTx := []*firefly.MappedTransaction{
		{
			Original: &database.Transaction{
				TransactionSource: "Bank",
				Date:              time.Now(),
			},
		},
	}
	errArr := []error{}

	result := p.Dry(ctx, mappedTx, errArr)
	assert.NotEmpty(t, result)
}

func TestPrinter_Duplicates(t *testing.T) {
	p := printer.NewPrinter()

	mappedTx := []*firefly.MappedTransaction{
		{
			Original: &database.Transaction{
				TransactionSource: "Bank",
				Date:              time.Now(),
			},
			Error: common.ErrDuplicate,
		},
	}

	result := p.Duplicates(context.Background(), mappedTx, nil)

	assert.Contains(t, result, "Duplicate: ✨")
}

func TestPrinter_Errors(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := printer.NewPrinter()

		mappedTx := []*firefly.MappedTransaction{
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("some error"),
			},
		}
		errArr := []error{errors.New("another error")}

		result := p.Errors(context.Background(), mappedTx, errArr)

		assert.Contains(t, result, "some error")
		assert.Contains(t, result, "another error")
	})

	t.Run("skip duplicates", func(t *testing.T) {
		mappedTx := []*firefly.MappedTransaction{
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("some error"),
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Duplicated",
					Date:              time.Now(),
				},
				Error: common.ErrDuplicate,
			},
		}

		errArr := []error{errors.New("another error")}

		p := printer.NewPrinter()
		result := p.Errors(context.Background(), mappedTx, errArr)

		assert.Contains(t, result, "some error")
		assert.Contains(t, result, "another error")
		assert.NotContains(t, result, "Duplicated")
	})

	t.Run("skip with no error", func(t *testing.T) {
		mappedTx := []*firefly.MappedTransaction{
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("some error"),
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Duplicated",
					Date:              time.Now(),
				},
			},
		}

		errArr := []error{errors.New("another error")}

		p := printer.NewPrinter()
		result := p.Errors(context.Background(), mappedTx, errArr)

		assert.Contains(t, result, "some error")
		assert.Contains(t, result, "another error")
		assert.NotContains(t, result, "Duplicated")
	})

	t.Run("no errors at all", func(t *testing.T) {
		mappedTx := []*firefly.MappedTransaction{
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
			},
		}

		errArr := []error{errors.New("another error")}

		p := printer.NewPrinter()
		result := p.Errors(context.Background(), mappedTx, errArr)

		assert.Contains(t, result, "No errors.")
	})
}

func TestPrinter_Stat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := printer.NewPrinter()

		mappedTx := []*firefly.MappedTransaction{
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: common.ErrDuplicate,
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: common.ErrOperationNotSupported,
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: common.ErrOperationNotSupported,
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("unknown error"),
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("unknown error"),
			},
			{
				Original: &database.Transaction{
					TransactionSource: "Bank",
					Date:              time.Now(),
				},
				Error: errors.New("unknown error"),
			},
		}

		var errArr []error

		result := p.Stat(context.Background(), mappedTx, errArr)

		assert.Contains(t, result, "Total transactions")
		assert.Contains(t, result, "Ok: 1 🔥")
		assert.Contains(t, result, "Errors: 3")
		assert.Contains(t, result, "Duplicates: 1")
		assert.Contains(t, result, "Unsupported operations: 2")
	})
}

func TestPrinter_fancyPrintTx(t *testing.T) {
	p := printer.NewPrinter()
	sb := &strings.Builder{}

	tx := &firefly.MappedTransaction{
		Original: &database.Transaction{
			TransactionSource: "Bank",
			Date:              time.Now(),
		},
	}

	p.FancyPrintTx(tx, sb)
	result := sb.String()

	assert.Contains(t, result, "Source: Bank")
}
