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

	err := duplicateCleaner.IsDuplicate(context.Background(), "", database.Zen)
	assert.NoError(t, err)
}

func TestIsDuplicate_RepoReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().IsDuplicateKeyExists(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, errors.New("repo error"))

	err := duplicateCleaner.IsDuplicate(context.Background(), "test-key", database.Zen)
	assert.Error(t, err)
	assert.Equal(t, "repo error", err.Error())
}

func TestIsDuplicate_KeyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().IsDuplicateKeyExists(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	err := duplicateCleaner.IsDuplicate(context.Background(), "test-key", database.Zen)
	assert.Error(t, err)
	assert.Error(t, duplicatecleaner.DuplicateTransactionError, err)
}

func TestIsDuplicate_KeyDoesNotExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockRepo(ctrl)
	duplicateCleaner := duplicatecleaner.NewDuplicateCleaner(mockRepo)

	mockRepo.EXPECT().IsDuplicateKeyExists(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)

	err := duplicateCleaner.IsDuplicate(context.Background(), "test-key", database.Zen)
	assert.NoError(t, err)
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
