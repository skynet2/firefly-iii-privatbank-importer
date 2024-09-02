package parser

import (
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Record struct {
	Message *database.Message
	Data    []byte
}
