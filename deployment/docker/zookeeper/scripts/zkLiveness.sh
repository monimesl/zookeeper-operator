#!/usr/bin/env bash

source /config/bootEnv.sh
source /scripts/zkCommon.sh

set -x
if [[ $(echo ruok | nc "$CLIENT_HOST" "$CLIENT_PORT") != "imok" ]]; then
  echo "The zookeeper node failed a liveliness check"
  exit 1
else
  exit 0
fi
