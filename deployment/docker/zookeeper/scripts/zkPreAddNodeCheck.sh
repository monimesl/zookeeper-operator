#!/usr/bin/env bash

set +e

SERVICE_PID=$1

echo "Checking if the background process succeed" >&2
pgrep -P "$SERVICE_PID" &>/dev/null
if [[ $? -ne 0 ]]; then
  echo "The node failed to start!!" >&2
  exit 0
fi

echo "Checking if the service is alive" >&2
for ((i = 0; i < 3; ++i)); do
  /scripts/zkLiveness.sh
  if [[ $? -eq 0 ]]; then
    exit 0
  fi
  sleep 1
done

echo "The node failed to start!!" >&2
exit 1
