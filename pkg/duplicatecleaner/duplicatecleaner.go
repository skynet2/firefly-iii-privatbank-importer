package duplicatecleaner

import (
	"context"
	"crypto/sha512"
	"fmt"

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

func (d *DuplicateCleaner) GetDuplicates(
	ctx context.Context,
	keys []string,
	txSource database.TransactionSource,
) (map[string]struct{}, error) {
	var hashedKeys []string
	for _, key := range keys {
		if key == "" {
			continue
		}

		hashedKeys = append(hashedKeys, d.HashKey(key))
	}

	final := map[string]struct{}{}

	if len(hashedKeys) == 0 {
		return final, nil
	}

	exists, err := d.repo.GetDuplicates(ctx, hashedKeys, txSource)
	if err != nil {
		return nil, err
	}

	for _, key := range exists {
		final[key] = struct{}{}
	}

	return final, nil
}

func (d *DuplicateCleaner) AddDuplicateKey(
	ctx context.Context,
	key string,
	txSource database.TransactionSource,
) error {
	if key == "" {
		return nil
	}

	key = d.HashKey(key)

	return d.repo.AddDuplicateKey(ctx, key, txSource)
}

func (d *DuplicateCleaner) HashKey(bv string) string {
	shaImpl := sha512.New()
	shaImpl.Write([]byte(bv))

	return fmt.Sprintf("%x", shaImpl.Sum(nil))
}
