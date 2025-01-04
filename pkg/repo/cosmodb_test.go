package repo_test

import (
	"context"
	"crypto/sha512"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/repo"
)

func TestCosmo(t *testing.T) {
	client, err := azcosmos.NewClientFromConnectionString(os.Getenv("COSMO_DB_CONNECTION_STRING"), nil)
	if err != nil {
		panic(err)
	}

	local, err := repo.NewCosmo(client, "test")
	assert.NoError(t, err)

	key := hash(time.Now().UTC().Format(time.RFC3339Nano))

	ok, err := local.GetDuplicates(context.TODO(), []string{
		"d3664f14e04723c64a4116d2a7710235f7c3ac1348e62747b085772a92e32f96c63acdbd078c35f3e217fabb1fc140281e35fb628aa135ff367b8e8fdaef5bea",
		"b",
	}, database.Paribas)
	assert.NoError(t, err)
	assert.NotNil(t, ok)

	err = local.AddDuplicateKey(context.TODO(), key, database.Paribas)
	assert.NoError(t, err)

	ok, err = local.GetDuplicates(context.TODO(), []string{key}, database.Paribas)
	assert.NoError(t, err)
	assert.NotNil(t, ok)
}

func hash(bv string) string {
	shaImpl := sha512.New()
	shaImpl.Write([]byte(bv))

	return fmt.Sprintf("%x", shaImpl.Sum(nil))
}
