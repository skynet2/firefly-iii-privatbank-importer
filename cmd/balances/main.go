package main

import (
	"os"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/imroc/req/v3"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	ffData, err := fetchAccountData()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch account data")
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("POSTGRES_CONNECTION_STRING")), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get postgres")
	}

	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "gorm_migrations",
		IDColumnName:              "id",
		IDColumnSize:              255,
		UseTransaction:            false,
		ValidateUnknownMigrations: false,
	}, getMigrations())

	log.Info().Msg("[Db] start migrations")

	if err = m.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to migrate")
	}

	dbData, err := fetchDbData(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch db data")
	}

	tx := db.Begin()
	defer tx.Rollback()

	for _, account := range dbData {
		_, ok := ffData[account.ID]
		if !ok { // drop accounts which does not exist in Firefly
			if err = tx.Delete(&account).Error; err != nil {
				log.Fatal().Err(err).Msg("failed to delete account")
			}
		}
	}

	for _, account := range ffData {
		dbAccount := dbData[account.ID]

		if dbAccount.ID == 0 || !dbAccount.Balance.Equal(account.Balance) || dbAccount.CurrencyID != account.CurrencyID {
			dbAccount = account
			dbAccount.UpdatedAt = time.Now().UTC()

			if err = tx.Clauses(clause.OnConflict{UpdateAll: true}).Save(&dbAccount).Error; err != nil {
				log.Fatal().Err(err).Msg("failed to save account")
			}
		}
	}

	if err = tx.Commit().Error; err != nil {
		log.Fatal().Err(err).Msg("failed to commit transaction")
	}
}

func fetchDbData(db *gorm.DB) (map[int]simpleAccountData, error) {
	var records []simpleAccountData

	if err := db.Find(&records).Error; err != nil {
		return nil, errors.Wrap(err, "failed to fetch records")
	}

	accountData := map[int]simpleAccountData{}
	for _, record := range records {
		accountData[record.ID] = record
	}

	return accountData, nil
}

func fetchAccountData() (map[int]simpleAccountData, error) {
	ffURL := os.Getenv("FIREFLY_API_ENDPOINT")

	httpClient := req.DefaultClient()
	request := httpClient.R().SetHeader("Authorization", "Bearer "+os.Getenv("FIREFLY_API_KEY"))

	var result genericResponse[[]accountResponse]

	res, err := request.SetSuccessResult(&result).Get(ffURL)
	if err != nil {
		return nil, err
	}

	if res.IsErrorState() {
		return nil, errors.Newf("failed to make request: body=%s", res.String())
	}

	accountData := map[int]simpleAccountData{}
	for _, account := range result.Data {
		if !account.Attributes.Active {
			continue
		}

		parsedID, parseErr := strconv.Atoi(account.ID)
		if parseErr != nil {
			return nil, errors.Newf("failed to parse account ID: %s", account.ID)
		}

		pasedCurrency, parseErr := strconv.Atoi(account.Attributes.CurrencyID)
		if parseErr != nil {
			return nil, errors.Newf("failed to parse currency ID: %s", account.Attributes.CurrencyID)
		}

		accountData[parsedID] = simpleAccountData{
			ID:         parsedID,
			Balance:    account.Attributes.CurrentBalance,
			CurrencyID: pasedCurrency,
		}
	}

	return accountData, nil
}
