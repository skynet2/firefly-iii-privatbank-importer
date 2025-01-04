package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/common"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
)

type Printer struct {
}

func NewPrinter() *Printer {
	return &Printer{}
}

func (p *Printer) Commit(
	ctx context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
) string {
	return p.Dry(ctx, mappedTx, errArr)
}

func (p *Printer) Dry(
	ctx context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
) string {
	var sb strings.Builder

	sb.WriteString(p.Stat(ctx, mappedTx, errArr))
	sb.WriteString("\n\n")

	for _, tx := range mappedTx {
		if errors.Is(tx.Error, common.ErrDuplicate) {
			continue
		}
		if errors.Is(tx.Error, common.ErrOperationNotSupported) {
			continue
		}

		p.FancyPrintTx(tx, &sb)
	}

	return sb.String()
}

func (p *Printer) Duplicates(
	_ context.Context,
	mappedTx []*firefly.MappedTransaction,
	_ []error,
) string {
	var duplicates []*firefly.MappedTransaction

	for _, tx := range mappedTx {
		if errors.Is(tx.Error, common.ErrDuplicate) {
			duplicates = append(duplicates, tx)
		}
	}

	if len(duplicates) == 0 {
		return "No duplicates found"
	}

	var sb strings.Builder
	for _, tx := range duplicates {
		p.FancyPrintTx(tx, &sb)
	}

	if len(duplicates) == len(mappedTx) {
		sb.WriteString("\nAll transactions are duplicates: ✅")
	}

	return sb.String()
}

func (p *Printer) Errors(
	_ context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
) string {
	var errCount int
	var sb strings.Builder

	for _, err := range errArr {
		sb.WriteString(fmt.Sprintf("Error: %s\n", err))
	}

	for _, tx := range mappedTx {
		if tx.Error == nil {
			continue
		}

		if errors.Is(tx.Error, common.ErrDuplicate) {
			continue
		}

		p.FancyPrintTx(tx, &sb)

		errCount += 1
	}

	if errCount == 0 {
		sb.WriteString("No errors.")
	}

	return sb.String()
}

func (p *Printer) Stat(
	_ context.Context,
	mappedTx []*firefly.MappedTransaction,
	errArr []error,
) string {
	var duplicateCount int
	var notSupportedCount int
	var okCount int

	for _, tx := range mappedTx {
		if tx.Error == nil {
			okCount += 1
			continue
		}

		if errors.Is(tx.Error, common.ErrDuplicate) {
			duplicateCount += 1
			continue
		}

		if errors.Is(tx.Error, common.ErrOperationNotSupported) {
			notSupportedCount += 1
			continue
		}

		errArr = append(errArr, tx.Error)
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Total transactions: %v", len(mappedTx)))
	sb.WriteString(fmt.Sprintf("\nOk: %v 🔥", okCount))

	sb.WriteString(fmt.Sprintf("\nErrors: %v 🚒", len(errArr)))
	sb.WriteString(fmt.Sprintf("\nUnsupported operations: %v 🚯", notSupportedCount))

	sb.WriteString(fmt.Sprintf("\nDuplicates: %v ✨", duplicateCount))

	if okCount == len(mappedTx) {
		sb.WriteString("\n\nAll transactions are ok! 🎉")
	}

	return sb.String()
}

func (p *Printer) FancyPrintTx(tx *firefly.MappedTransaction, sb *strings.Builder) {
	if tx.IsCommitted {
		sb.WriteString("Committed: ✅\n")
	}

	if tx.Error != nil {
		if errors.Is(tx.Error, common.ErrDuplicate) {
			sb.WriteString("Duplicate: ✨\n")
		} else {
			sb.WriteString("Has Error: ❌\n")
		}
	}

	sb.WriteString(fmt.Sprintf("Source: %v", tx.Original.TransactionSource))
	sb.WriteString(fmt.Sprintf("\nDate: %s\n", tx.Original.Date.Format("2006-01-02 15:04")))

	if !tx.Original.SourceAmount.IsZero() {
		sb.WriteString(fmt.Sprintf("\nSource: %v%v", tx.Original.SourceAmount.StringFixed(2), tx.Original.SourceCurrency))
	}
	if tx.Original.SourceAccount != "" {
		sb.WriteString(fmt.Sprintf("\nSource Account: %s", tx.Original.SourceAccount))
	}
	if tx.Transaction != nil && tx.Transaction.SourceName != "" {
		sb.WriteString(fmt.Sprintf("\nSource [FF]: %s", tx.Transaction.SourceName))
	}
	sb.WriteString("\n")

	if !tx.Original.DestinationAmount.IsZero() {
		sb.WriteString(fmt.Sprintf("\nDestination: %v%v",
			tx.Original.DestinationAmount.StringFixed(2), tx.Original.DestinationCurrency))
	}
	if tx.Original.DestinationAccount != "" {
		sb.WriteString(fmt.Sprintf("\nDestination Account: %s", tx.Original.DestinationAccount))
	}
	if tx.Transaction != nil && tx.Transaction.DestinationName != "" {
		sb.WriteString(fmt.Sprintf("\nDestination [FF]: %s", tx.Transaction.DestinationName))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("\nType: %v", tx.Original.Type))
	if tx.Transaction != nil {
		sb.WriteString(fmt.Sprintf("\nType [FF]: %s", tx.Transaction.Type))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("\nDescription: %s", tx.Original.Description))
	//sb.WriteString(fmt.Sprintf("\nDuplication Key: %s", strings.Join(tx.Original.DeduplicationKeys, "")))

	if tx.Error != nil {
		sb.WriteString(fmt.Sprintf("\nERROR: %s", tx.Error))
	}

	sb.WriteString("\n====================\n")
}
