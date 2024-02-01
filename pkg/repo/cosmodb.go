package repo

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/cockroachdb/errors"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	messagesContainer = "messages"
)

type Cosmo struct {
	cl *azcosmos.DatabaseClient
}

func NewCosmo(
	cl *azcosmos.Client,
	dbName string,
) (*Cosmo, error) {
	_, err := cl.CreateDatabase(context.Background(), azcosmos.DatabaseProperties{
		ID: dbName,
	}, &azcosmos.CreateDatabaseOptions{})

	c := &Cosmo{}

	if realErr := c.ignoreDuplicateErr(err); realErr != nil {
		return nil, realErr
	}

	db, err := cl.NewDatabase(dbName)
	if err != nil {
		return nil, err
	}
	c.cl = db

	if err = c.setupContainers(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cosmo) setupContainers() error {
	_, err := c.cl.CreateContainer(context.Background(), azcosmos.ContainerProperties{
		ID: messagesContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/isProcessed"},
		},
	}, &azcosmos.CreateContainerOptions{})

	return c.ignoreDuplicateErr(err)
}

func (c *Cosmo) ignoreDuplicateErr(err error) error {
	if err == nil {
		return nil
	}
	var azureErr *azcore.ResponseError
	if errors.As(err, &azureErr) && azureErr.StatusCode == 409 {
		return nil
	}

	return err
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
	if err := c.setupContainers(); err != nil {
		return nil, err
	}

	return c.cl.NewContainer(messagesContainer)
}

func (c *Cosmo) GetLatestMessages(ctx context.Context) ([]*database.Message, error) {
	container, err := c.getMessageContainer()
	if err != nil {
		return nil, err
	}

	partitionKey := azcosmos.NewPartitionKeyBool(false)

	query := "SELECT * FROM c where c.isProcessed = false order by c.createdAt desc"
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

func (s *Cosmo) UpdateMessage(ctx context.Context, message database.Message) error {
	container, err := s.getMessageContainer()
	if err != nil {
		return err
	}

	partitionKey := azcosmos.NewPartitionKeyBool(message.IsProcessed)

	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = container.UpsertItem(ctx, partitionKey, bytes, nil)

	return err
}
