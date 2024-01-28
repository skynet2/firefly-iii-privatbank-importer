package processor

import "context"

type Processor struct {
	repo Repo
}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) AddMessage(
	ctx context.Context,
	message string,
) {

}

func (p *Processor) ProcessLatestMessages(
	ctx context.Context,
) error {
	return nil
}
