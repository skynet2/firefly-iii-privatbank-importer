package processor_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	parser2 "github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

func TestProcessorCommit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := NewMockRepo(gomock.NewController(t))
		parser := NewMockParser(gomock.NewController(t))

		fireflySvc := NewMockFirefly(gomock.NewController(t))
		notificationSvc := NewMockNotificationSvc(gomock.NewController(t))

		srv := processor.NewProcessor(&processor.Config{
			Repo:            repo,
			NotificationSvc: notificationSvc,
			FireflySvc:      fireflySvc,
			Parsers: map[database.TransactionSource]processor.Parser{
				database.PrivatBank: parser,
			},
		})

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
				OriginalMessage: messages[0],
			},
			{
				OriginalMessage: messages[1],
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
