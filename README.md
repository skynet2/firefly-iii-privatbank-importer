# firefly-iii-importer

![build workflow](https://github.com/skynet2/firefly-iii-privatbank-importer/actions/workflows/general.yaml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/skynet2/firefly-iii-privatbank-importer/branch/master/graph/badge.svg?token=5QV4Z8NR6V)](https://codecov.io/gh/skynet2/firefly-iii-privatbank-importer)
[![go-report](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/skynet2/firefly-iii-privatbank-importer)](https://pkg.go.dev/github.com/skynet2/firefly-iii-privatbank-importer?tab=doc)


## Overview
The Firefly III Importer is a tool designed to automate the process of importing banking transactions from banks into Firefly III, a self-hosted personal finance manager. 

## Supported Banks

### PrivatBank (next.privat24.ua)
- Protocol: Telegram Notifications
- Supported Transaction Types: 
  - [x] Income
  - [x] Withdrawal
  - [x] Transfer
- [ ] Duplicate cleaner

### Paribas (goonline.bnpparibas.pl)
- Protocol: XLSX (statements export)
- Supported Transaction Types: 
  - [x] Income
  - [x] Withdrawal
  - [x] Transfer
- [ ] Duplicate cleaner

### MonoBank (monobank.ua)
- Protocol: CSV
- Supported Transaction Types: 
  - [ ] Income
  - [x] Withdrawal
  - [ ] Transfer
- [x] Duplicate cleaner

### Revolut (revolut.com)
- Protocol: CSV
- Supported Transaction Types: 
  - [ ] Income
  - [x] Withdrawal
  - [ ] Transfer
- [x] Duplicate cleaner

### Server deployment
1. cd cmd/server && go build -o server
2. deploy to your environment
3. Setup environment variables
```bash
export CHAT_MAP = {"<telegram_chat_id>" : "privatbank", "<telegram_chat_id_2>" : "paribas"}
export COSMO_DB_CONNECTION_STRING = "AccountEndpoint=..."
export COSMO_DB_NAME = "firefly-importer"
export FIREFLY_URL = "https://firefly.example.com"
export FIREFLY_TOKEN= "your_firefly_token"
export TELEGRAM_BOT_TOKEN = "your_telegram_bot_token"
export FIREFLY_ADDITIONAL_HEADERS = {"header1" : "val1", "header2" : "val2"}
```
4. Set telegram webhook url to your host (endpoint /api/github/webhook)

## Bot Usage
To use the Firefly III Importer, you need to set up a Telegram bot and connect it to your group.

### For PrivateBank
Forward Privat notifications to Importer group.

## Default Bot Commands
### /dry - Perform a dry run. This command processes all pending messages without committing the transactions to Firefly III.
### /commit - Commit all pending transactions to Firefly III. This command processes and imports all pending transactions.
### /clear - Clear all pending transactions. Use this command to remove any messages that you do not want to import.
