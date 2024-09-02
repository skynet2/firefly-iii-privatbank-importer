package repo_test

//
//import (
//	"context"
//	"crypto/sha512"
//	"fmt"
//	"os"
//	"testing"
//	"time"
//
//	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
//	"github.com/stretchr/testify/assert"
//
//	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
//	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/repo"
//)
//
//func TestCosmo(t *testing.T) {
//	client, err := azcosmos.NewClientFromConnectionString(os.Getenv("COSMO_DB_CONNECTION_STRING"), nil)
//	if err != nil {
//		panic(err)
//	}
//
//	local, err := repo.NewCosmo(client, "test")
//	assert.NoError(t, err)
//
//	key := hash(time.Now().UTC().Format(time.RFC3339Nano))
//
//	ok, err := local.IsDuplicateKeyExists(context.TODO(), key, database.Paribas)
//	assert.NoError(t, err)
//	assert.False(t, ok)
//
//	err = local.AddDuplicateKey(context.TODO(), key, database.Paribas)
//	assert.NoError(t, err)
//
//	ok, err = local.IsDuplicateKeyExists(context.TODO(), key, database.Paribas)
//	assert.NoError(t, err)
//	assert.True(t, ok)
//}
//
//func hash(bv string) string {
//	shaImpl := sha512.New()
//	shaImpl.Write([]byte(bv))
//
//	return fmt.Sprintf("%x", shaImpl.Sum(nil))
//}
