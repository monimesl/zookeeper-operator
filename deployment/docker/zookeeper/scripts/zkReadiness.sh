#!/usr/bin/env bash

source /config/bootEnv.sh
source /scripts/zkCommon.sh

set -x
nslookup "$SERVICE_NAME" &>/dev/null
if [[ $? -eq 1 ]]; then
  echo "The ensemble service \"$SERVICE_NAME\" is not available"
  exit 0
else
  set -e
  if [[ $(echo ruok | nc "$CLIENT_HOST" "$CLIENT_PORT") != "imok" ]]; then
    echo "The zookeeper node failed a readiness check"
    exit 1
  else
    set +e
    ## @Todo: Add membership checking
    exit 0
  fi
fi
