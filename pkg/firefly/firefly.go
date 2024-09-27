package firefly

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req/v3"
	"github.com/shopspring/decimal"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Firefly struct {
	cl                *req.Client
	apiKey            string
	fireflyURL        string
	additionalHeaders map[string]string
}

func NewFirefly(
	apiKey string,
	fireflyURL string,
	cl *req.Client,
	additionalHeaders map[string]string,
) *Firefly {
	return &Firefly{
		cl:                cl,
		fireflyURL:        fireflyURL,
		apiKey:            apiKey,
		additionalHeaders: additionalHeaders,
	}
}

func (f *Firefly) getBaseRequest(ctx context.Context) *req.Request {
	baseReq := f.cl.R().
		SetContext(ctx).
		SetBearerAuthToken(f.apiKey)

	for k, v := range f.additionalHeaders {
		baseReq = baseReq.SetHeader(k, v)
	}

	return baseReq
}

func (f *Firefly) ListAccounts(ctx context.Context) ([]*Account, error) {
	var apiResp GenericApiResponse[[]*Account]

	resp, err := f.getBaseRequest(ctx).
		SetSuccessResult(&apiResp).
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
			Error:    tx.ParsingError,
		}

		finalTransactions = append(finalTransactions, mapped)

		if mapped.Error != nil { // pass-through
			continue
		}
		switch tx.Type {
		case database.TransactionTypeRemoteTransfer:
			fallthrough
		case database.TransactionTypeExpense:
			acc, ok := accountByAccountNumber[tx.SourceAccount]
			if !ok {
				mapped.Error = errors.Newf("account with IBAN %s not found", tx.SourceAccount)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:                "withdrawal",
				Date:                tx.Date.Format(time.RFC3339),
				Amount:              tx.SourceAmount.StringFixed(2),
				Description:         tx.Description,
				CurrencyCode:        tx.SourceCurrency,
				SourceID:            acc.Id,
				SourceName:          acc.Attributes.Name,
				Notes:               tx.Raw,
				ForeignCurrencyCode: tx.DestinationCurrency,
			}

			if tx.DestinationAmount.GreaterThan(decimal.Zero) {
				mapped.Transaction.ForeignAmount = tx.DestinationAmount.StringFixed(2)
			}

			if tx.DestinationAccount != "" {
				if dst, dstOk := accountByAccountNumber[tx.DestinationAccount]; dstOk {
					mapped.Transaction.DestinationID = dst.Id
					mapped.Transaction.DestinationName = dst.Attributes.Name
				}
			}
		case database.TransactionTypeInternalTransfer:
			sourceID := tx.SourceAccount
			destinationID := tx.DestinationAccount

			accSource, ok := accountByAccountNumber[sourceID]
			if !ok {
				mapped.Error = errors.Newf("source account with IBAN %s not found", sourceID)
				continue
			}

			accDestination, ok := accountByAccountNumber[destinationID]
			if !ok {
				mapped.Error = errors.Newf("destination account with IBAN %s not found", destinationID)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:                "transfer",
				Date:                tx.Date.Format(time.RFC3339),
				Amount:              tx.SourceAmount.StringFixed(2),
				Description:         tx.Description,
				CurrencyCode:        tx.SourceCurrency,
				SourceID:            accSource.Id,
				SourceName:          accSource.Attributes.Name,
				DestinationID:       accDestination.Id,
				DestinationName:     accDestination.Attributes.Name,
				Notes:               tx.Raw,
				ForeignAmount:       tx.DestinationAmount.StringFixed(2),
				ForeignCurrencyCode: tx.DestinationCurrency,
			}
		case database.TransactionTypeIncome:
			acc, ok := accountByAccountNumber[tx.DestinationAccount]
			if !ok {
				mapped.Error = errors.Newf("account with IBAN %s not found", tx.DestinationAccount)
				continue
			}

			mapped.Transaction = &Transaction{
				Type:            "deposit",
				Date:            tx.Date.Format(time.RFC3339),
				Amount:          tx.DestinationAmount.StringFixed(2),
				Description:     tx.Description,
				CurrencyCode:    tx.DestinationCurrency,
				DestinationID:   acc.Id,
				DestinationName: acc.Attributes.Name,
				Notes:           tx.Raw,
			}

			if tx.SourceAccount != "" {
				if src, sourceOk := accountByAccountNumber[tx.SourceAccount]; sourceOk {
					mapped.Transaction.SourceID = src.Id
					mapped.Transaction.SourceName = src.Attributes.Name
					mapped.Transaction.ForeignCurrencyCode = tx.SourceCurrency
					mapped.Transaction.ForeignAmount = tx.SourceAmount.StringFixed(2)
				}
			}
		default:
			mapped.Error = errors.Newf("unknown transaction type %d", tx.Type)
		}
	}

	return finalTransactions, nil
}

func (f *Firefly) CreateTransactions(
	ctx context.Context,
	tx *Transaction,
	errorOnDuplicate bool,
) (*Transaction, error) {
	var apiResp GenericApiResponse[Transaction]

	resp, err := f.getBaseRequest(ctx).
		SetSuccessResult(&apiResp).
		SetHeader("Accept", "application/json").
		SetBody(map[string]interface{}{
			"apply_rules":             true,
			"error_if_duplicate_hash": errorOnDuplicate,
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
