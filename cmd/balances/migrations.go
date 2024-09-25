package main

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func getMigrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "2024_06_09_Initial",
			Migrate: func(db *gorm.DB) error {
				return db.Exec(`create table if not exists simple_account_data_importer
(
    id          integer not null
        constraint simple_account_data_importer_pk
            primary key,
    balance     decimal,
    currency_id integer
);
`).Error
			},
		},
		{
			ID: "2024_06_09_AddUpdatedAt",
			Migrate: func(db *gorm.DB) error {
				return db.Exec(`alter table simple_account_data_importer
	add updated_at timestamp;
`).Error
			},
		},
		{
			ID: "2024_09_25_AddDailyStats",
			Migrate: func(db *gorm.DB) error {
				return db.Exec(`create table if not exists simple_account_data_importer_daily
(
    id          integer not null,
    balance     decimal,
    currency_id integer,
    date date,
    updated_at timestamp,
    constraint simple_account_data_importer_daily_pl
        primary key (id, date)
);
`).Error
			},
		},
	}
}
