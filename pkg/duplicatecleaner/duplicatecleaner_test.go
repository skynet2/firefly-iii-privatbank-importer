package duplicatecleaner_test

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/duplicatecleaner"
)

func TestIsDuplicate_KeyIsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	duplicates, err := duplicateCleaner.GetDuplicates(context.Background(), []string{""}, database.Zen)
	assert.NoError(t, err)
	assert.Empty(t, duplicates)
}

func TestIsDuplicate_RepoReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().GetDuplicates(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("repo error"))

	_, err := duplicateCleaner.GetDuplicates(context.Background(), []string{"test-key"}, database.Zen)
	assert.Error(t, err)
	assert.Equal(t, "repo error", err.Error())
}

func TestIsDuplicate_KeyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().GetDuplicates(gomock.Any(), gomock.Any(), gomock.Any()).Return([]string{
		"test-key",
	}, nil)

	duplicates, err := duplicateCleaner.GetDuplicates(context.Background(), []string{"test-key"}, database.Zen)

	assert.NoError(t, err)
	assert.Len(t, duplicates, 1)
	assert.Contains(t, duplicates, "test-key")
}

func TestIsDuplicate_KeyDoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().GetDuplicates(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	results, err := duplicateCleaner.GetDuplicates(context.Background(), []string{"test-key"}, database.Zen)
	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.NotNil(t, results)
}

func TestAddDuplicateKey_KeyIsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	err := duplicateCleaner.AddDuplicateKey(context.Background(), "", database.Zen)
	assert.NoError(t, err)
}

func TestAddDuplicateKey_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().AddDuplicateKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	err := duplicateCleaner.AddDuplicateKey(context.Background(), "test-key", database.Zen)
	assert.NoError(t, err)
}

func TestAddDuplicateKey_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().AddDuplicateKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("repo error"))

	err := duplicateCleaner.AddDuplicateKey(context.Background(), "test-key", database.Zen)
	assert.Error(t, err)
	assert.Equal(t, "repo error", err.Error())
}
