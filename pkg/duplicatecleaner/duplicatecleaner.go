package duplicatecleaner

import (
	"context"
	"crypto/sha512"
	"fmt"

	"github.com/cockroachdb/errors"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type DuplicateCleaner struct {
	repo Repo
}

func NewDuplicateCleaner(
	repo Repo,
) *DuplicateCleaner {
	return &DuplicateCleaner{
		repo: repo,
	}
}

func (d *DuplicateCleaner) IsDuplicate(
	ctx context.Context,
	key string,
	txSource database.TransactionSource,
) error {
	if key == "" {
		return nil
	}

	key = d.hash(key)
	exists, err := d.repo.IsDuplicateKeyExists(ctx, key, txSource)
	if err != nil {
		return err
	}

	if exists {
		return errors.WithStack(DuplicateTransactionError)
	}

	return nil
}

func (d *DuplicateCleaner) AddDuplicateKey(
	ctx context.Context,
	key string,
	txSource database.TransactionSource,
) error {
	if key == "" {
		return nil
	}

	key = d.hash(key)

	return d.repo.AddDuplicateKey(ctx, key, txSource)
}

func (d *DuplicateCleaner) hash(bv string) string {
	shaImpl := sha512.New()
	shaImpl.Write([]byte(bv))

	return fmt.Sprintf("%x", shaImpl.Sum(nil))
}
