package common

import "github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"

type ChatConfiguration struct {
	Source          database.TransactionSource
	SkipDuplicates  bool
	SkipIncomeError bool
}
