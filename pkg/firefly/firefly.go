package firefly

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req/v3"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Firefly struct {
	cl         *req.Client
	apiKey     string
	fireflyURL string
}

func NewFirefly(
	apiKey string,
	fireflyURL string,
	cl *req.Client,
) *Firefly {
	return &Firefly{
		cl:         cl,
		fireflyURL: fireflyURL,
		apiKey:     apiKey,
	}
}

func (f *Firefly) ListAccounts(ctx context.Context) ([]*Account, error) {
	var apiResp GenericApiResponse[[]*Account]

	resp, err := f.cl.R().
		SetContext(ctx).
		SetBearerAuthToken(f.apiKey).
		SetSuccessResult(&apiResp).
		EnableDumpTo(os.Stdout).
		SetHeader("Accept", "application/json").
		SetQueryParam("limit", "100500").
		Get(f.fireflyURL + "/api/v1/accounts")
	if err != nil {
		return nil, err
	}

	if resp.IsErrorState() {
		return nil, errors.Newf("got error response: %s", resp.String())
	}

	return apiResp.Data, nil
}

func (f *Firefly) MapTransactions(
	ctx context.Context,
	transactions []*database.Transaction,
) ([]*MappedTransaction, error) {
	accounts, err := f.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	accountByAccountNumber := map[string]*Account{}
	for _, acc := range accounts {
		sp := strings.Split(acc.Attributes.AccountNumber, ",")
		if len(sp) == 0 {
			continue
		}

		for _, s := range sp {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}

			_, ok := accountByAccountNumber[s]
			if ok {
				return nil, errors.Newf("duplicate account number %s", s)
			}

			accountByAccountNumber[s] = acc
		}
	}

	var finalTransactions []*MappedTransaction
	for _, tx := range transactions {
		mapped := &MappedTransaction{
			Original: tx,
		}

		finalTransactions = append(finalTransactions, mapped)
		switch tx.Type {
		case database.TransactionTypeRemoteTransfer:
			fallthrough
		case database.TransactionTypeExpense:
			acc, ok := accountByAccountNumber[tx.SourceAccount]
			if !ok {
				mapped.MappingError = errors.Newf("account with IBAN %s not found", tx.SourceAccount)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:         "withdrawal",
				Date:         tx.Date.Format(time.RFC3339),
				Amount:       tx.SourceAmount.String(),
				Description:  tx.Description,
				CurrencyCode: tx.SourceCurrency,
				SourceID:     acc.Id,
				SourceName:   acc.Attributes.Name,
				Notes:        tx.Raw,
			}
		case database.TransactionTypeInternalTransfer:
			sourceID := tx.SourceAccount
			destinationID := tx.DestinationAccount

			accSource, ok := accountByAccountNumber[sourceID]
			if !ok {
				mapped.MappingError = errors.Newf("source account with IBAN %s not found", sourceID)
				continue
			}

			accDestination, ok := accountByAccountNumber[destinationID]
			if !ok {
				mapped.MappingError = errors.Newf("destination account with IBAN %s not found", destinationID)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:                "transfer",
				Date:                tx.Date.Format(time.RFC3339),
				Amount:              tx.SourceAmount.String(),
				Description:         tx.Description,
				CurrencyCode:        tx.SourceCurrency,
				SourceID:            accSource.Id,
				SourceName:          accSource.Attributes.Name,
				DestinationID:       accDestination.Id,
				DestinationName:     accDestination.Attributes.Name,
				Notes:               tx.Raw,
				ForeignAmount:       tx.DestinationAmount.String(),
				ForeignCurrencyCode: tx.DestinationCurrency,
			}
		case database.TransactionTypeIncome:
			acc, ok := accountByAccountNumber[tx.DestinationAccount]
			if !ok {
				mapped.MappingError = errors.Newf("account with IBAN %s not found", tx.DestinationAccount)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:            "deposit",
				Date:            tx.Date.Format(time.RFC3339),
				Amount:          tx.DestinationAmount.String(),
				Description:     tx.Description,
				CurrencyCode:    tx.DestinationCurrency,
				DestinationID:   acc.Id,
				DestinationName: acc.Attributes.Name,
				Notes:           tx.Raw,
			}
		default:
			mapped.MappingError = errors.Newf("unknown transaction type %d", tx.Type)
		}
	}

	return finalTransactions, nil
}

func (f *Firefly) CreateTransactions(ctx context.Context, tx *Transaction) (*Transaction, error) {
	var apiResp GenericApiResponse[Transaction]

	resp, err := f.cl.R().
		SetContext(ctx).
		SetBearerAuthToken(f.apiKey).
		SetSuccessResult(&apiResp).
		EnableDumpTo(os.Stdout).
		SetHeader("Accept", "application/json").
		SetBody(map[string]interface{}{
			"apply_rules":             true,
			"error_if_duplicate_hash": false,
			"fire_webhooks":           true,
			"transactions":            []*Transaction{tx},
		}).
		Post(f.fireflyURL + "/api/v1/transactions")
	if err != nil {
		return nil, err
	}

	if resp.IsErrorState() {
		return nil, errors.Newf("got error response: %s", resp.String())
	}

	return &apiResp.Data, nil
}
