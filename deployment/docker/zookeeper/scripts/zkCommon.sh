#!/usr/bin/env bash

RETRIES=20

POD_NAME=`hostname -s`
CONFIG_DIR=$DATA_DIR/conf
MYID_FILE=$DATA_DIR/myid
STATIC_CONFIG_FILE=$CONFIG_DIR/zoo.cfg
DYNAMIC_CONFIG_FILE=$CONFIG_DIR/zoo.cfg.dynamic

function zkServerConfig() {
  role=$1
  HOST="$POD_NAME.$SERVICE_NAME"
  echo "$HOST:$QUORUM_PORT:$LEADER_PORT:$role;$CLIENT_PORT"
}

function zkConnectionString() {
  set +e
  nslookup $CLIENT_HOST &>/dev/null
  if [[ $? -eq 0 ]]; then
    set -e
    echo "$CLIENT_HOST:$CLIENT_PORT"
  else
    retries=$RETRIES
    while [ $retries -gt 0 ]
    do
      sleep 2
      echo "zkConnectionString() retry countdown: $retries" >&2
      if [[ $(nslookup "$CLIENT_HOST" &>/dev/null) ]]; then
        echo "$CLIENT_HOST:$CLIENT_PORT"
        return
      fi
      retries=$((retries - 1))
    done
    set -e
    echo "zkConnectionString() failed: unable to lookup client host($CLIENT_HOST)"
    exit 1
  fi
}

function ensemblePresent() {
  set +e
  ## Check if there is already an existing ensemble
  LOOKUP_RESULT=$(nslookup $SERVICE_NAME)
  if [[ $? -eq 0 ]]; then
    checkServicePort
    return $?
#  elif echo $LOOKUP_RESULT | grep -q "server can't find $SERVICE_NAME"; then
#    echo "could not detect any existing ensemble:: $LOOKUP_RESULT ::"
#    return 1
  else ## lookup failed; do a sleep-then retry loop for a finite time
    retries=$RETRIES
    while [ $retries -gt 0 ]
    do
      sleep 2
      retries=$((retries - 1))
      echo "ensemblePresent() retry-countdown: $retries"
      if [[ $(nslookup $SERVICE_NAME) -eq 0 ]]; then
        checkServicePort
        return $?
      fi
    done
    return 1
  fi
}

function checkServicePort() {
  set +e
  nc -z -w5 $SERVICE_NAME $CLIENT_PORT
  if [[ $? -ne 0 ]]; then
     retries=$RETRIES
     while [ $retries -gt 0 ]
     do
       sleep 2
       retries=$((retries - 1))
       echo "checkServicePort() retry-countdown: $retries" >&2
       nc -z -w5 $SERVICE_NAME $CLIENT_PORT
       if [[ $? -eq 0 ]]; then
         return 0
       fi
     done
     return 1
  fi
  return 0
}