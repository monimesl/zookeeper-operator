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

set -x +e

MYID=$(cat "$MYID_FILE")
ZK_URL=$(zkClientUrl)

echo "Removing the node readiness file: $NODE_READY_FILE"
rm -f "$NODE_READY_FILE"

echo "Syncing and fetching the size of the cluster $CLUSTER_NAME"
SIZE=""
for ((i = 0; i < 15; i++)); do
  SYNC=$(zk-shell "$ZK_URL" --run-once "sync $CLUSTER_META_SIZE_NODE_PATH")
  if [[ -z "${SYNC}" ]]; then
    SIZE=$(zk-shell "$ZK_URL" --run-once "get $CLUSTER_META_SIZE_NODE_PATH")
    break
  fi
  echo "Failed to connect. Retrying($i) after 2 seconds"
  SIZE=""
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
  # See `Progress guarantees` https://zookeeper.apache.org/doc/r3.6.3/zookeeperReconfig.html#ch_reconfig_dyn
  zk-shell "$ZK_URL" --run-once "set $CLUSTER_META_UPDATE_TIME_NODE_PATH '$(($(date +%s%N) / 1000000))'"
fi

# Wait the server to drain it's remote client connections
echo "Waiting the server to drain it's remote client connections"
for ((i = 0; i < 5; i++)); do
  CONNECTION_COUNT=$(echo cons | nc localhost 2181 | grep -cv "/127.0.0.1:")
  if [[ "$CONNECTION_COUNT" -gt 0 ]]; then
    echo "$CONNECTION_COUNT remote connection(s) still connected. Waiting counter: $i"
    sleep 1
  elif [[ "$CONNECTION_COUNT" -eq 0 ]]; then
    echo "The remote connections are completely drained!!"
    break
  else
    echo "Tired of waiting. Continuing shutdown with $CONNECTION_COUNT remote connection(s) still connected"
    break
  fi
done

## cleanup the config files so on next restart/cluster join we can recreate them
echo "Cleaning up the config files so on next restart/cluster join we can recreate them"
rm "$DYNAMIC_CONFIG_FILE" "$STATIC_CONFIG_FILE"

echo "Eager kill the process instead of waiting for kubernetes 'TerminationGracePeriodSeconds'"

lsof -i :"$CLIENT_PORT" | grep LISTEN | awk '{print $2}' | xargs kill 2>/dev/null
