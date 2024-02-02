# firefly-iii-privatbank-importer

![build workflow](https://github.com/skynet2/firefly-iii-privatbank-importer/actions/workflows/build.yaml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/skynet2/firefly-iii-privatbank-importer/branch/master/graph/badge.svg?token=5QV4Z8NR6V)](https://codecov.io/gh/skynet2/firefly-iii-privatbank-importer)
[![go-report](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/skynet2/firefly-iii-privatbank-importer)](https://pkg.go.dev/github.com/skynet2/firefly-iii-privatbank-importer?tab=doc)


## Overview
The Firefly III PrivatBank Importer is a tool designed to automate the process of importing banking transactions from PrivatBank into Firefly III, a self-hosted personal finance manager. Utilizing Telegram as an intermediary, this application captures transaction notifications from PrivatBank sent to a user's Telegram account and imports them into Firefly III, simplifying the management of personal finances.

## Features
Automatic Transaction Import: Seamlessly import transactions from PrivatBank into Firefly III.
Telegram Bot Integration: Use a dedicated Telegram bot to process and forward banking notifications.
Account and Currency Mapping: Automatically map PrivatBank accounts and currencies to their corresponding entities in Firefly III.
Dry Run Mode: Preview transactions before committing them to Firefly III, ensuring accuracy.
Commit and Clear Commands: Easily commit pending transactions or clear them if necessary.


## Usage
To use the Firefly III PrivatBank Importer, you need to set up a Telegram bot and connect it to your PrivatBank account to receive transaction notifications. Once set up, forward these notifications to your dedicated Telegram bot.

## Default Bot Commands
### /dry - Perform a dry run. This command processes all pending messages without committing the transactions to Firefly III.
### /commit - Commit all pending transactions to Firefly III. This command processes and imports all pending transactions.
### /clear - Clear all pending transactions. Use this command to remove any messages that you do not want to import.
