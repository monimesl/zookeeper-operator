#!/bin/bash
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

VERSION=$1
DOCKER_IMAGE=monime/zookeeper-operator:latest
if [[ -n $VERSION ]]; then
  HELM_VERSION="${VERSION//v/}" #helm version must be a SemVer
  DOCKER_IMAGE="monime/zookeeper-operator:$VERSION"
  sed -i "s|image:.*|image: $DOCKER_IMAGE|" deployments/charts/operator/values.yaml
  sed -i "s|version:.*|version: $HELM_VERSION|; s|appVersion:.*|appVersion: $HELM_VERSION|" deployments/charts/operator/Chart.yaml
fi

echo "apiVersion: v1
kind: Namespace
metadata:
   name: zookeeper-operator" >deployments/operator-manifest.yaml

helm template default --include-crds --namespace zookeeper-operator deployments/charts/operator/ >>deployments/operator-manifest.yaml

sed -i "/app.kubernetes.io\/managed-by: Helm/d" deployments/operator-manifest.yaml
