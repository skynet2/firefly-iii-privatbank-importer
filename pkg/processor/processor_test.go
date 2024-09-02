package processor_test

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	parser2 "github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

func TestDuplicateMessage(t *testing.T) {
	t.Run("one duplicate tx", func(t *testing.T) {
		repo := NewMockRepo(gomock.NewController(t))
		parser := NewMockParser(gomock.NewController(t))

		fireflySvc := NewMockFirefly(gomock.NewController(t))
		notificationSvc := NewMockNotificationSvc(gomock.NewController(t))

		dedup := NewMockDuplicateCleaner(gomock.NewController(t))

		srv := processor.NewProcessor(&processor.Config{
			Repo:             repo,
			DuplicateCleaner: dedup,
			NotificationSvc:  notificationSvc,
			FireflySvc:       fireflySvc,
			Parsers: map[database.TransactionSource]processor.Parser{
				database.PrivatBank: parser,
			},
		})

		dedup.EXPECT().IsDuplicate(gomock.Any(), "111", database.PrivatBank).
			Return(errors.New("is duplicate"))
		dedup.EXPECT().IsDuplicate(gomock.Any(), "1234", database.PrivatBank).
			Return(nil)

		messages := []*database.Message{
			{
				ChatID:    1234,
				MessageID: 4321,
			},
			{
				ChatID:    1234,
				MessageID: 4321,
			},
		}
		resultTxs := []*database.Transaction{
			{
				OriginalMessage:  messages[0],
				DeduplicationKey: "1234",
			},
			{
				OriginalMessage:  messages[1],
				DeduplicationKey: "111",
			},
		}

		fireflyTxs := []*firefly.MappedTransaction{
			{
				Original: resultTxs[0],
			},
			{
				Original: resultTxs[1],
			},
		}

		fireflySvc.EXPECT().MapTransactions(gomock.Any(), resultTxs).
			Return([]*firefly.MappedTransaction{
				{
					Original: resultTxs[0],
				},
				{
					Original: resultTxs[1],
				},
			}, nil)

		fireflySvc.EXPECT().CreateTransactions(gomock.Any(), fireflyTxs[0].Transaction).
			Return(&firefly.Transaction{}, nil)

		repo.EXPECT().UpdateMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, messages []*database.Message) error {
				assert.Len(t, messages, 2)
				return nil
			})

		parser.EXPECT().ParseMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, records []*parser2.Record) ([]*database.Transaction, error) {
				return resultTxs, nil
			})

		repo.EXPECT().GetLatestMessages(gomock.Any(), database.PrivatBank).
			Return(messages, nil)

		notificationSvc.EXPECT().React(gomock.Any(), int64(1234), int64(4321), "🍾").
			Return(nil)

		notificationSvc.EXPECT().SendMessage(gomock.Any(), int64(111), gomock.Any()).
			DoAndReturn(func(ctx context.Context, i int64, s string) error {
				assert.Contains(t, s, "Duplicates: 1")
				return nil
			})

		notificationSvc.EXPECT().SendMessage(gomock.Any(), int64(111), gomock.Any()).
			DoAndReturn(func(ctx context.Context, i int64, s string) error {
				assert.Contains(t, s, "All Ok")
				assert.Contains(t, s, "Duplicates: 1")
				return nil
			})

		assert.NoError(t, srv.Commit(context.TODO(), processor.Message{
			TransactionSource: database.PrivatBank,
			ChatID:            111,
		}))
	})

	t.Run("all duplicate tx", func(t *testing.T) {
		repo := NewMockRepo(gomock.NewController(t))
		parser := NewMockParser(gomock.NewController(t))

		fireflySvc := NewMockFirefly(gomock.NewController(t))
		notificationSvc := NewMockNotificationSvc(gomock.NewController(t))

		dedup := NewMockDuplicateCleaner(gomock.NewController(t))

		srv := processor.NewProcessor(&processor.Config{
			Repo:             repo,
			DuplicateCleaner: dedup,
			NotificationSvc:  notificationSvc,
			FireflySvc:       fireflySvc,
			Parsers: map[database.TransactionSource]processor.Parser{
				database.PrivatBank: parser,
			},
		})

		dedup.EXPECT().IsDuplicate(gomock.Any(), "111", database.PrivatBank).
			Return(errors.New("is duplicate"))
		dedup.EXPECT().IsDuplicate(gomock.Any(), "1234", database.PrivatBank).
			Return(errors.New("is duplicate"))

		messages := []*database.Message{
			{
				ChatID:    1234,
				MessageID: 4321,
			},
			{
				ChatID:    1234,
				MessageID: 4321,
			},
		}
		resultTxs := []*database.Transaction{
			{
				OriginalMessage:  messages[0],
				DeduplicationKey: "1234",
			},
			{
				OriginalMessage:  messages[1],
				DeduplicationKey: "111",
			},
		}

		fireflySvc.EXPECT().MapTransactions(gomock.Any(), resultTxs).
			Return([]*firefly.MappedTransaction{
				{
					Original: resultTxs[0],
				},
				{
					Original: resultTxs[1],
				},
			}, nil)

		repo.EXPECT().UpdateMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, messages []*database.Message) error {
				assert.Len(t, messages, 2)
				return nil
			})

		parser.EXPECT().ParseMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, records []*parser2.Record) ([]*database.Transaction, error) {
				return resultTxs, nil
			})

		repo.EXPECT().GetLatestMessages(gomock.Any(), database.PrivatBank).
			Return(messages, nil)

		//notificationSvc.EXPECT().React(gomock.Any(), int64(1234), int64(4321), "🍾").
		//	Return(nil)

		notificationSvc.EXPECT().SendMessage(gomock.Any(), int64(111), gomock.Any()).
			DoAndReturn(func(ctx context.Context, i int64, s string) error {
				assert.Contains(t, s, "Duplicates: 2")
				return nil
			})

		assert.NoError(t, srv.Commit(context.TODO(), processor.Message{
			TransactionSource: database.PrivatBank,
			ChatID:            111,
		}))
	})
}

func TestProcessorCommit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepo(gomock.NewController(t))
		parser := NewMockParser(gomock.NewController(t))

		fireflySvc := NewMockFirefly(gomock.NewController(t))
		notificationSvc := NewMockNotificationSvc(gomock.NewController(t))

		dedup := NewMockDuplicateCleaner(gomock.NewController(t))

		srv := processor.NewProcessor(&processor.Config{
			Repo:             repo,
			DuplicateCleaner: dedup,
			NotificationSvc:  notificationSvc,
			FireflySvc:       fireflySvc,
			Parsers: map[database.TransactionSource]processor.Parser{
				database.PrivatBank: parser,
			},
		})

		dedup.EXPECT().IsDuplicate(gomock.Any(), "111", database.PrivatBank).
			Return(nil)
		dedup.EXPECT().IsDuplicate(gomock.Any(), "1234", database.PrivatBank).
			Return(nil)

		messages := []*database.Message{
			{
				ChatID:    1234,
				MessageID: 4321,
			},
			{
				ChatID:    1234,
				MessageID: 4321,
			},
		}
		resultTxs := []*database.Transaction{
			{
				OriginalMessage:  messages[0],
				DeduplicationKey: "1234",
			},
			{
				OriginalMessage:  messages[1],
				DeduplicationKey: "111",
			},
		}

		fireflyTxs := []*firefly.MappedTransaction{
			{
				Original: resultTxs[0],
			},
			{
				Original: resultTxs[1],
			},
		}

		fireflySvc.EXPECT().MapTransactions(gomock.Any(), resultTxs).
			Return([]*firefly.MappedTransaction{
				{
					Original: resultTxs[0],
				},
				{
					Original: resultTxs[1],
				},
			}, nil)

		fireflySvc.EXPECT().CreateTransactions(gomock.Any(), fireflyTxs[0].Transaction).
			Return(&firefly.Transaction{}, nil)
		fireflySvc.EXPECT().CreateTransactions(gomock.Any(), fireflyTxs[1].Transaction).
			Return(&firefly.Transaction{}, nil)

		repo.EXPECT().UpdateMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, messages []*database.Message) error {
				assert.Len(t, messages, 2)
				return nil
			})

		parser.EXPECT().ParseMessages(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, records []*parser2.Record) ([]*database.Transaction, error) {
				return resultTxs, nil
			})

		repo.EXPECT().GetLatestMessages(gomock.Any(), database.PrivatBank).
			Return(messages, nil)

		notificationSvc.EXPECT().React(gomock.Any(), int64(1234), int64(4321), "🍾").
			Return(nil)

		notificationSvc.EXPECT().SendMessage(gomock.Any(), int64(111), gomock.Any()).
			Return(nil)

		assert.NoError(t, srv.Commit(context.TODO(), processor.Message{
			TransactionSource: database.PrivatBank,
			ChatID:            111,
		}))
	})
}
