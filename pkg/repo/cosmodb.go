package repo

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/cockroachdb/errors"
	"github.com/gammazero/workerpool"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	messagesContainer = "messages"
	defaultPoolSize   = 50
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
			Paths: []string{"/transactionSource"},
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

func (c *Cosmo) AddMessage(ctx context.Context, messages []database.Message) error {
	if len(messages) == 0 {
		return nil
	}

	container, err := c.getMessageContainer()
	if err != nil {
		return err
	}

	pool := workerpool.New(defaultPoolSize)

	var finalErr error

	for _, msg1 := range messages {
		msgCopy := msg1

		pool.Submit(func() {
			if finalErr != nil {
				return
			}

			partitionKey := azcosmos.NewPartitionKeyString(string(msgCopy.TransactionSource))
			bytes, msgErr := json.Marshal(msgCopy)
			if msgErr != nil {
				finalErr = errors.Join(finalErr, msgErr)
				return
			}

			_, err = container.CreateItem(ctx, partitionKey, bytes, nil)
			if err != nil {
				finalErr = errors.Join(finalErr, err)
				return
			}
		})
	}

	pool.StopWait()

	return nil
}

func (c *Cosmo) getMessageContainer() (*azcosmos.ContainerClient, error) {
	if err := c.setupContainers(); err != nil {
		return nil, err
	}

	return c.cl.NewContainer(messagesContainer)
}

func (c *Cosmo) GetLatestMessages(
	ctx context.Context,
	transactionSource database.TransactionSource,
) ([]*database.Message, error) {
	container, err := c.getMessageContainer()
	if err != nil {
		return nil, err
	}

	partitionKey := azcosmos.NewPartitionKeyString(string(transactionSource))

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

func (c *Cosmo) Clear(ctx context.Context, transactionSource database.TransactionSource) error {
	container, err := c.getMessageContainer()
	if err != nil {
		return err
	}

	msg, err := c.GetLatestMessages(ctx, transactionSource)
	if err != nil {
		return err
	}

	pool := workerpool.New(defaultPoolSize)

	for _, m1 := range msg {
		copyMsg := m1

		pool.Submit(func() {
			if _, deleteErr := container.DeleteItem(
				ctx,
				azcosmos.NewPartitionKeyString(string(transactionSource)), copyMsg.ID, nil); deleteErr != nil {
				err = errors.Join(err, deleteErr)
			}
		})
	}

	pool.StopWait()

	return err
}

func (c *Cosmo) UpdateMessages(ctx context.Context, messages []*database.Message) error {
	container, err := c.getMessageContainer()
	if err != nil {
		return err
	}

	pool := workerpool.New(defaultPoolSize)
	for _, ms1 := range messages {
		msCopy := ms1

		pool.Submit(func() {
			partitionKey := azcosmos.NewPartitionKeyString(string(msCopy.TransactionSource))
			bytes, msgErr := json.Marshal(msCopy)
			if msgErr != nil {
				err = errors.Join(err, msgErr)
				return
			}

			_, err = container.UpsertItem(ctx, partitionKey, bytes, nil)
			if err != nil {
				err = errors.Join(err, err)
			}
		})
	}

	pool.StopWait()

	return err
}
