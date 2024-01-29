package repo

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	messagesContainer = "messages"
)

type Cosmo struct {
	cl *azcosmos.DatabaseClient
}

func (c *Cosmo) AddMessage(ctx context.Context, message database.Message) error {
	partitionKey := azcosmos.NewPartitionKeyBool(message.IsProcessed)

	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	container, err := c.getMessageContainer()

	_, err = container.CreateItem(ctx, partitionKey, bytes, nil)
	if err != nil {
		return err
	}

	return nil
}
func (c *Cosmo) getMessageContainer() (*azcosmos.ContainerClient, error) {
	return c.cl.NewContainer(messagesContainer)
}

func (c *Cosmo) GetLatestMessages(ctx context.Context) ([]*database.Message, error) {
	container, err := c.getMessageContainer()
	if err != nil {
		return nil, err
	}

	partitionKey := azcosmos.NewPartitionKeyBool(false)

	query := "SELECT * FROM messages where is_processed = false order by created_at desc"
	pager := container.NewQueryItemsPager(query, partitionKey, nil)

	var items []*database.Message

	for pager.More() {
		response, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			return nil, pageErr
		}

		for _, bytes := range response.Items {
			item := database.Message{}
			err = json.Unmarshal(bytes, &item)
			if err != nil {
				return nil, err
			}

			items = append(items, &item)
		}
	}

	return items, nil
}

func (c *Cosmo) Clear(ctx context.Context) error {
	container, err := c.getMessageContainer()
	if err != nil {
		return err
	}

	_, err = container.Delete(ctx, &azcosmos.DeleteContainerOptions{})

	return err
}

func NewCosmo(cl *azcosmos.DatabaseClient) *Cosmo {
	return &Cosmo{
		cl: cl,
	}
}
