#!/bin/sh

SLEEP_TIMEOUT=${SLEEP_TIMEOUT:=5}

while true; do
  ./balances
  sleep $SLEEP_TIMEOUT
done