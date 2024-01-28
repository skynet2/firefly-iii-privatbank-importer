package repo

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/syumai/workers/cloudflare/d1"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type D1 struct {
	dbName    string
	connector driver.Connector
}

func NewD1(dbName string) (*D1, error) {
	c, err := d1.OpenConnector(context.Background(), dbName)
	if err != nil {
		return nil, err
	}

	return &D1{
		dbName:    dbName,
		connector: c,
	}, nil
}

func (d *D1) AddMessage(
	ctx context.Context,
	message database.Message,
) error {
	db := sql.OpenDB(d.connector)
	defer func() {
		_ = db.Close()
	}()

	_, err := db.ExecContext(ctx, "INSERT INTO messages (id, created_at, processed_at, content) VALUES (?,?,?,?)",
		message.ID, message.CreatedAt, message.ProcessedAt, message.Content)

	return err
}

func (d *D1) Clear(ctx context.Context) error {
	db := sql.OpenDB(d.connector)
	defer func() {
		_ = db.Close()
	}()

	_, err := db.ExecContext(ctx, "DELETE FROM messages")

	return err
}

func (d *D1) GetLatestMessages(ctx context.Context) ([]*database.Message, error) {
	db := sql.OpenDB(d.connector)
	defer func() {
		_ = db.Close()
	}()

	rows, err := db.QueryContext(ctx, "SELECT id, created_at, processed_at, content FROM messages ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var messages []*database.Message
	for rows.Next() {
		var message database.Message
		if err = rows.Scan(&message.ID, &message.CreatedAt, &message.ProcessedAt, &message.Content); err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}

	return messages, nil
}
