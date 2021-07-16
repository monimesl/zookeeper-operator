#!/bin/bash
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

VERSION=$1
HELM_PACKAGE_DIR="${2:-output}"
DOCKER_IMAGE=monime/zookeeper-operator:latest
if [[ -n $VERSION ]]; then
  HELM_VERSION="${VERSION//v/}" #helm version must be a SemVer
  DOCKER_IMAGE="monime/zookeeper-operator:$VERSION"
  sed -i "s|image:.*|image: $DOCKER_IMAGE|" deployments/charts/operator/values.yaml
  sed -i "s|version:.*|version: $HELM_VERSION|; s|appVersion:.*|appVersion: $HELM_VERSION|" deployments/charts/operator/Chart.yaml
fi

cp deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml webhook.temp
cat config/webhook/manifests.yaml >>deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml
sed -i "/clientConfig:/a \    caBundle: {{ \$caBundle }}" deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml
sed -i "s|namespace: system|namespace: {{ .Release.Namespace }}|" deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml
sed -i 's|name: webhook-service|name: {{ include "operator.webhook-service" . }}|' deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml

OPERATOR_NAMESPACE=zookeeper-operator

printf "Generating the manifests\n"
make manifests
printf "Copying the CRDs files to the chart\n"
mkdir -p deployments/charts/operator/crds &&
  cp -r config/crd/bases/* deployments/charts/operator/crds
printf "Packaging the Helm chart\n"
helm package deployments/charts/operator/ -d "$HELM_PACKAGE_DIR"

printf "Generating the operator installation manifest\n"
helm template zookeeper-operator --include-crds --namespace $OPERATOR_NAMESPACE deployments/charts/operator/ >deployments/manifest.yaml

echo -e "# create the namespace
---
apiVersion: v1
kind: Namespace
metadata:
   name: $OPERATOR_NAMESPACE\n$(cat deployments/manifest.yaml)" >deployments/manifest.yaml

printf "Cleaning up\n"
cp -f webhook.temp deployments/charts/operator/templates/webhookSecretAndConfigurations.yaml &&
  rm -r webhook.temp deployments/charts/operator/crds
