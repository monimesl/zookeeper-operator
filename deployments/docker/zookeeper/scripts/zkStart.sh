#!/usr/bin/env bash

#
# Copyright 2020 - now, the original author or authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#       https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

source /config/bootEnv.sh
source /scripts/zkCommon.sh

#enable job control
set -m
set -ex

mkdir -p "$CONFIG_DIR"

# Extract resource name and this members ordinal value from the pod's hostname
if [[ $POD_NAME =~ (.*)-([0-9]+)$ ]]; then
  NAME=${BASH_REMATCH[1]}
  MYID=$((BASH_REMATCH[2] + 1))
else
  echo "bad hostname \"$POD_NAME\". Expecting to match the regex: (.*)-([0-9]+)$"
  exit 1
fi

MYID_FILE_PRESENT=false
DYNAMIC_CONFIG_FILE_PRESENT=false

if [ -f "$MYID_FILE" ]; then
  CURRENT_ID=$(cat "$MYID_FILE")
  if [[ "$CURRENT_ID" == "$MYID" && -f $STATIC_CONFIG_FILE ]]; then
    # If Id is correct and configuration is present under `/data/conf`
    MYID_FILE_PRESENT=true
  fi
fi

if [ -f "$DYNAMIC_CONFIG_FILE" ]; then
  set +e
  # shellcheck disable=SC2086
  # shellcheck disable=SC2002
  cat $DYNAMIC_CONFIG_FILE | grep -q "server.${MYID}="
  if [[ $? -eq 0 ]]; then
    DYNAMIC_CONFIG_FILE_PRESENT=true
  fi
fi

ADD_NODE=true

set +e
checkEnsemblePresence $MYID
if [[ $? -ne 0 ]]; then
  echo "Couldn't detect an ensemble; this may be the first node or the ensemble service in unavailable"
  ADD_NODE=false
elif [[ "$MYID_FILE_PRESENT" == true && "$DYNAMIC_CONFIG_FILE_PRESENT" == true ]]; then
  echo "This node is already a member of the ensemble"
  ADD_NODE=false
fi
set -e

SERVER_CONFIG=""
if [[ "$MYID_FILE_PRESENT" == false || "$DYNAMIC_CONFIG_FILE_PRESENT" == false ]]; then
  echo "Node configuration is missing; writing myid: $MYID to: $MYID_FILE"
  echo $MYID >"$MYID_FILE"
  if [[ $MYID -eq 1 ]]; then
    ADD_NODE=false
    echo "I'm the first server pod in the statefulset. Generating my dynamic config..."
    SERVER_CONFIG="server.${MYID}=$(zkServerConfig participant)"
    echo "Writing my dynamic configuration to $DYNAMIC_CONFIG_FILE"
    echo "$SERVER_CONFIG"
    echo "$SERVER_CONFIG" >"$DYNAMIC_CONFIG_FILE"
  else
    echo "I'm a subsequent server pod in the statefulset. Retrieving the current ensemble config..."
    ZK_URL=$(zkClientUrl)
    SERVER_CONFIG="server.${MYID}=$(zkServerConfig observer)"
    DYNAMIC_CONFIG=$(zk-shell "$ZK_URL" --run-once "get /zookeeper/config" | cat | head -n -1)
    DYNAMIC_CONFIG+="\n$SERVER_CONFIG"
    echo "Writing my dynamic configuration to $DYNAMIC_CONFIG_FILE"
    echo -en "$DYNAMIC_CONFIG"
    echo -en "$DYNAMIC_CONFIG" >"$DYNAMIC_CONFIG_FILE"
  fi
fi

if [[ ! -f $STATIC_CONFIG_FILE ]]; then
  echo "The static config file does not exists. copying /conf/zoo.cfg to $CONFIG_DIR"
  cp -f /config/zoo.cfg "$CONFIG_DIR"
fi

cp -f /config/log4j.properties "$CONFIG_DIR"
cp -f /config/log4j-quiet.properties "$CONFIG_DIR"

ZOOCFGDIR=$CONFIG_DIR
export ZOOCFGDIR

echo "Starting the zookeeper service in the background"
/zk/bin/zkServer.sh --config "$ZOOCFGDIR" start-foreground &
SERVICE_PID=$!
SERVICE_JOB=$(jobs -l | grep $SERVICE_PID | cut -d"[" -f2 | cut -d"]" -f1)

if [[ "$ADD_NODE" == false ]]; then
  # put the process back into foreground
  fg "$SERVICE_JOB"
else
  set +e
  sleep 1
  /scripts/zkPreAddNodeCheck.sh $SERVICE_PID
  if [[ $? -eq 0 ]]; then
    set -e
    ZK_URL=$(zkClientUrl)
    echo "Adding the node to the ensemble"
    DYNAMIC_CONFIG=$(zk-shell "$ZK_URL" --run-once "reconfig add $SERVER_CONFIG")
    if ! echo "$DYNAMIC_CONFIG" | grep -q "$SERVER_CONFIG"; then
      echo "Unable to add the node to the ensemble. See error below:"
      echo "$DYNAMIC_CONFIG"
      exit 1
    else
      echo "The node is successfully added to the ensemble"
      # put the process back into foreground
      fg "$SERVICE_JOB"
    fi
  else
    exit 1
  fi
fi
