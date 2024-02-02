package main

import (
	"context"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

type MessageProcessor interface {
	ProcessMessage(
		ctx context.Context,
		message processor.Message,
	) error
}
