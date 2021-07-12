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
