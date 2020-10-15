#!/usr/bin/env bash

source /config/bootEnv.sh
source /scripts/zkCommon.sh

set -ex

mkdir -p $CONFIG_DIR

# Extract resource name and this members ordinal value from the pod's hostname
if [[ $POD_NAME =~ (.*)-([0-9]+)$ ]]; then
    NAME=${BASH_REMATCH[1]}
    MYID=$((${BASH_REMATCH[2]} + 1))
else
    echo "bad hostname \"$POD_NAME\". Expecting to match the regex: (.*)-([0-9]+)$"
    exit 1
fi

MYID_FILE_PRESENT=false
DYNAMIC_CONFIG_FILE_PRESENT=false

if [ -f $MYID_FILE ]; then
  CURRENT_ID=$(cat $MYID_FILE)
    if [[ "$CURRENT_ID" == "$MYID" && -f $STATIC_CONFIG_FILE ]]; then
    # If Id is correct and configuration is present under `/data/conf`
      MYID_FILE_PRESENT=true
  fi
fi

if [ -f $DYNAMIC_CONFIG_FILE ]; then
  DYNAMIC_CONFIG_FILE_PRESENT=true
fi



ENSEMBLE_PRESENT=false

set +e
$(ensemblePresent)
if [[ $? -eq 0 ]]; then
  echo "An ensemble is already existing at '$SERVICE_NAME'"
  ENSEMBLE_PRESENT=true
fi
set -e

ADD_NODE=false

if [[ "$MYID_FILE_PRESENT" == false || "$DYNAMIC_CONFIG_FILE_PRESENT" == false ]]; then
    echo "Node configuration is missing; writing myid: $MYID to: $MYID_FILE"
    echo $MYID > $MYID_FILE
    if [[ $MYID -eq 1 ]]; then
      echo "I'm the first server in the ensemble; my dynamic config will be generated from the local template."
      DYNAMIC_CONFIG="server.${MYID}=$(zkServerConfig participant)"
      echo "Writing my dynamic configuration to $DYNAMIC_CONFIG_FILE"
      echo $DYNAMIC_CONFIG
      echo $DYNAMIC_CONFIG > $DYNAMIC_CONFIG_FILE
    else
      set -e
      echo "There is already an ensemble; my dynamic config will be created from the ensemble's configuration."
      ZK_URL=$(zkConnectionString)
      DYNAMIC_CONFIG=$(zk-shell $ZK_URL --run-once "get /zookeeper/config" | cat | head -n -1)
      echo "Writing my dynamic configuration to $DYNAMIC_CONFIG_FILE"
      echo "$DYNAMIC_CONFIG"
      echo "$DYNAMIC_CONFIG" > $DYNAMIC_CONFIG_FILE
      set +e
    fi
  if [[ "$ENSEMBLE_PRESENT" == true ]]; then
    ADD_NODE=true
  fi
fi

if [[ "$ADD_NODE" == true ]]; then
    set -e
    ZK_URL=$(zkConnectionString)
    DYNAMIC_CONFIG=$(zkServerConfig observer)
    echo "Adding the node to the ensemble and writing it's dynamic configuration to disk."
    DYNAMIC_CONFIG=$(zk-shell $ZK_URL --run-once "reconfig add $DYNAMIC_CONFIG" | cat | head -n -1)
    echo "Writing my dynamic configuration to $DYNAMIC_CONFIG_FILE"
    echo "$DYNAMIC_CONFIG"
    echo "$DYNAMIC_CONFIG" > $DYNAMIC_CONFIG_FILE
    set +e
fi

if [[ ! -f $STATIC_CONFIG_FILE ]]; then
  echo "The static config file does not exists. copying /conf/zoo.cfg to $CONFIG_DIR"
  cp -f /config/zoo.cfg $CONFIG_DIR
fi

cp -f /config/log4j.properties $CONFIG_DIR
cp -f /config/log4j-quiet.properties $CONFIG_DIR

ZOOCFGDIR=$CONFIG_DIR
export ZOOCFGDIR

if [ -f $DYNAMIC_CONFIG_FILE ]; then
  # This node was added to the ensemble
  echo "Starting the zookeeper service"
  /zk/bin/zkServer.sh --config $ZOOCFGDIR start-foreground
else
  echo "Zookeeper node setup failed!!"
  exit 1
fi