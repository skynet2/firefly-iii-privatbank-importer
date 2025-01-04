package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/cenkalti/backoff/v5"
	"github.com/cockroachdb/errors"
	"github.com/gammazero/workerpool"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

const (
	messagesContainer  = "messages"
	duplicateContainer = "duplicates"
	defaultPoolSize    = 10
)

type Cosmo struct {
	cl          *azcosmos.DatabaseClient
	setupCalled bool
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
	if c.setupCalled {
		return nil
	}

	_, err := c.cl.CreateContainer(context.Background(), azcosmos.ContainerProperties{
		ID: messagesContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/transactionSource"},
		},
	}, &azcosmos.CreateContainerOptions{})
	if c.ignoreDuplicateErr(err) != nil {
		return err
	}

	_, err = c.cl.CreateContainer(context.Background(), azcosmos.ContainerProperties{
		ID: duplicateContainer,
		PartitionKeyDefinition: azcosmos.PartitionKeyDefinition{
			Paths: []string{"/transactionSource"},
		},
	}, &azcosmos.CreateContainerOptions{})
	if c.ignoreDuplicateErr(err) != nil {
		return err
	}

	c.setupCalled = true

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

func (c *Cosmo) getRetryParams() []backoff.RetryOption {
	return []backoff.RetryOption{
		backoff.WithMaxTries(10),
		backoff.WithMaxElapsedTime(10 * time.Second),
		backoff.WithBackOff(backoff.NewExponentialBackOff()),
	}
}

func (c *Cosmo) getMessageContainer() (*azcosmos.ContainerClient, error) {
	if err := c.setupContainers(); err != nil {
		return nil, err
	}

	return c.cl.NewContainer(messagesContainer)
}

func (c *Cosmo) getDuplicateContainer() (*azcosmos.ContainerClient, error) {
	if err := c.setupContainers(); err != nil {
		return nil, err
	}

	return c.cl.NewContainer(duplicateContainer)
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

			_, err = backoff.Retry(ctx, func() (azcosmos.ItemResponse, error) {
				return container.CreateItem(ctx, partitionKey, bytes, nil)
			}, c.getRetryParams()...)

			if err != nil {
				finalErr = errors.Join(finalErr, err)
				return
			}
		})
	}

	pool.StopWait()

	return finalErr
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
			if _, deleteErr := backoff.Retry(ctx, func() (azcosmos.ItemResponse, error) {
				return container.DeleteItem(
					ctx,
					azcosmos.NewPartitionKeyString(string(transactionSource)), copyMsg.ID, nil)
			}, c.getRetryParams()...); deleteErr != nil {
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

			_, err = backoff.Retry(ctx, func() (azcosmos.ItemResponse, error) {
				return container.UpsertItem(ctx, partitionKey, bytes, nil)
			}, c.getRetryParams()...)

			if err != nil {
				err = errors.Join(err, err)
			}
		})
	}

	pool.StopWait()

	return err
}

type duplicateKey struct {
	ID                string `json:"id"`
	CreatedAt         string `json:"createdAt"`
	TransactionSource string `json:"transactionSource"`
}

func (c *Cosmo) AddDuplicateKey(
	ctx context.Context,
	key string,
	source database.TransactionSource,
) error {
	container, err := c.getDuplicateContainer()
	if err != nil {
		return err
	}

	partitionKey := azcosmos.NewPartitionKeyString(string(source))

	b, err := json.Marshal(duplicateKey{
		ID:                key,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		TransactionSource: string(source),
	})
	if err != nil {
		return err
	}

	_, err = backoff.Retry(ctx, func() (azcosmos.ItemResponse, error) {
		return container.CreateItem(ctx, partitionKey, b, nil)
	}, c.getRetryParams()...)

	return err
}

func (c *Cosmo) GetDuplicates(
	ctx context.Context,
	keys []string,
	source database.TransactionSource,
) ([]string, error) {
	container, err := c.getDuplicateContainer()
	if err != nil {
		return nil, err
	}

	partitionKey := azcosmos.NewPartitionKeyString(string(source))

	query := "SELECT c.id FROM c where ARRAY_CONTAINS(@id, c.id)"
	parameters := []azcosmos.QueryParameter{
		{
			Name:  "@id",
			Value: keys,
		},
	}

	pager := container.NewQueryItemsPager(query, partitionKey, &azcosmos.QueryOptions{
		QueryParameters: parameters,
	})

	var existing []string

	for pager.More() {
		response, pageErr := pager.NextPage(ctx)
		if pageErr != nil {
			return nil, pageErr
		}

		for _, item := range response.Items {
			var parsed duplicateKey
			if err = json.Unmarshal(item, &parsed); err != nil {
				return nil, err
			}

			existing = append(existing, parsed.ID)
		}
	}

	return existing, nil
}
