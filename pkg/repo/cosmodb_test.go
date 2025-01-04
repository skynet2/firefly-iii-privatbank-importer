package repo_test

//
//func TestCosmo(t *testing.T) {
//	client, err := azcosmos.NewClientFromConnectionString(os.Getenv("COSMO_DB_CONNECTION_STRING"), nil)
//	if err != nil {
//		panic(err)
//	}
//
//	local, err := repo.NewCosmo(client, "firefly-importer")
//	assert.NoError(t, err)
//
//	key := hash(time.Now().UTC().Format(time.RFC3339Nano))
//
//	ok, err := local.GetDuplicates(context.TODO(), []string{
//		hash("12323"),
//		"b",
//	}, database.Paribas)
//	assert.NoError(t, err)
//	assert.NotNil(t, ok)
//
//	err = local.AddDuplicateKey(context.TODO(), key, database.Paribas)
//	assert.NoError(t, err)
//
//	ok, err = local.GetDuplicates(context.TODO(), []string{key}, database.Paribas)
//	assert.NoError(t, err)
//	assert.NotNil(t, ok)
//}
//
//func hash(bv string) string {
//	shaImpl := sha512.New()
//	shaImpl.Write([]byte(bv))
//
//	return fmt.Sprintf("%x", shaImpl.Sum(nil))
//}
