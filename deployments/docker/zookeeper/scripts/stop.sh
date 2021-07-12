#!/usr/bin/env bash

#
# Copyright 2021 - now, the original author or authors.
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

source /scripts/common.sh

set -x

CLUSTER_META_NODE_PATH="$CLUSTER_METADATA_PARENT_ZNODE/$CLUSTER_NAME"

set +e
MYID=$(cat "$MYID_FILE")
ZK_URL=$(zkClientUrl)

set +e
echo "Syncing and fetching the size of the cluster $CLUSTER_NAME"
SIZE=""
for ((i = 0; i < 15; i++)); do
  SYNC=$(zk-shell "$ZK_URL" --run-once "sync $CLUSTER_META_NODE_PATH")
  if [[ -z "${SYNC}" ]]; then
    SIZE=$(zk-shell "$ZK_URL" --run-once "get $CLUSTER_META_NODE_PATH" | cut -d"=" -f2)
    break
  fi
  echo "Failed to connect. Retrying($i) after 2 seconds"
  SIZE=""
  sleep 2
done

echo "Cluster current SIZE=$SIZE, myid=$MYID"

# Since we're using kubernetes statefulset to start the node in an ordered fashion,
# the cluster size at any arbitrary normal point in time equals the highest `myid`.
# which is 1 increment of the ordinal of the pod running the container. On cluster
# down scaling($SIZE reduction), the pod with the highest ordinal hence `myid` is deleted.
# This means any node whose `myid` is greater than the current cluster size is being
# permanently removed from the ensemble
if [[ -n "$SIZE" && "$MYID" -gt "$SIZE" ]]; then
  echo "Removing the node with id $MYID from the cluster: $CLUSTER_NAME"
  zk-shell "$ZK_URL" --run-once "reconfig remove $MYID"
  # Ensure a quorum has activated the new configuration.
  # See `Progress guarantees` https://zookeeper.apache.org/doc/r3.6.2/zookeeperReconfig.html#ch_reconfig_dyn
  zk-shell "$ZK_URL" --run-once "set $CLUSTER_METADATA_PARENT_ZNODE/last-removal-time '$(date)'"
fi

# Wait the server to drain it's remote client connections
echo "Waiting the server to drain it's remote client connections"
for ((i = 0; i < 15; i++)); do
  CONNECTION_COUNT=$(echo cons | nc localhost 2181 | grep -cv "/127.0.0.1:")
  if [[ "$CONNECTION_COUNT" -gt 0 ]]; then
    echo "$CONNECTION_COUNT remote connection(s) still connected. Waiting for another 2 seconds"
    sleep 2
  elif [[ "$CONNECTION_COUNT" -eq 0 ]]; then
    echo "The remote connections are completely drained!!"
    break
  else
    echo "Tired of waiting. Continuing shutdown with $CONNECTION_COUNT remote connection(s) still connected"
    break
  fi
  echo "Waiting count-down: $i"
done

## cleanup the config files so on next restart/cluster join we can recreate them
echo "Cleaning up the config files so on next restart/cluster join we can recreate them"
rm "$DYNAMIC_CONFIG_FILE" "$STATIC_CONFIG_FILE"

echo "Eager kill the process instead of waiting for kubernetes 'TerminationGracePeriodSeconds'"

lsof -i :"$CLIENT_PORT" | grep LISTEN | awk '{print $2}' | xargs kill 2>/dev/nul
lsof -i :"$CLIENT_PORT" | grep LISTEN | awk '{print $2}' | xargs kill 2>/dev/nul
