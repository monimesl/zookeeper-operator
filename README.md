[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/monimesl/zookeeper-operator)](https://goreportcard.com/report/github.com/monimesl/zookeeper-operator)
[![Actions Status](https://github.com/monimesl/zookeeper-operator/workflows/Test%20and%20Build/badge.svg)](https://github.com/monimesl/zookeeper-operator/actions)

# Apache Zookeeper Operator

**Status: *alpha***

Simplify [Zookeeper](https://zookeeper.apache.org/) installation and management in kubernetes using CRDs

## Overview

The Zookeeper Operator enable native [Kubernetes](https://kubernetes.io/)
deployment and management of Apache Zookeeper Ensemble. To set up the cluster, the operator
uses [Zookeeper Dynamic Configuration](https://zookeeper.apache.org/doc/current/zookeeperReconfig.html)
which is supported by version __3.5+__ .
View [versions](https://github.com/monimesl/zookeeper-operator/blob/main/deployments/docker/zookeeper/versions)
to see the Zookeeper versions we provide support for. For now,
version [3.6.3_](https://www.apache.org/dyn/closer.lua/zookeeper/zookeeper-3.6.3/apache-zookeeper-3.6.3-bin.tar.gz) is
used as the installed version.

## Prerequisites

The operator needs a kubernetes cluster with a version __>= v1.16.0__ . If you're using [Helm](https://helm.sh/) to
install the operator, your helm version must be __>= 3.0.0__ .

## Installation

The operator can be installed and upgrade by using
our [helm chart](https://github.com/monimesl/zookeeper-operator/tree/main/deployments/charts)
or directly using
the [manifest file](https://github.com/monimesl/zookeeper-operator/blob/main/deployments/manifest.yaml). We however do
recommend using the [helm chart](https://github.com/monimesl/zookeeper-operator/tree/main/deployments/charts)
.

### Via [Helm](https://helm.sh/)

First you need to add the chart's [repository](https://monimesl.github.io/helm-charts/) to your repo list:

```bash
helm repo add monimesl https://monimesl.github.io/helm-charts
helm repo update
```

Create the operator namespace; we're doing this because Helm 3 no longer automatically create namespace.

```bash
kubectl create namespace zookeeper-operator
```

Now install the chart in the created namespace:

```bash
helm install zookeeper-operator monimesl/zookeeper-operator -n zookeeper-operator
```

### Via [Manifest](https://github.com/monimesl/zookeeper-operator/blob/main/deployments/manifest.yaml)

If you don't have [Helm](https://helm.sh/) or its required version, or you just want to try the operator quickly, this
option is then ideal. We provide a manifest file per operator version. The below command will install the latest
version.

Install the latest tag version:

```bash
 kubectl apply -f https://raw.githubusercontent.com/monimesl/zookeeper-operator/main/deployments/manifest.yaml
```

Or install the other tagged version you want by using the url below; replace `<tag-here>` with the tag.

```bash
 kubectl apply -f https://raw.githubusercontent.com/monimesl/zookeeper-operator/<tag-here>/deployments/manifest.yaml
```

Mind you, the command above will install a
[CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
and create a [ClusterRole](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/); so
the user issuing the command must have cluster-admin privileges.

#### Confirm Installation

Before continuing, ensure the operator pod is __ready__

```bash
kubectl wait --for=condition=ready --timeout=60s pod -l app.kubernetes.io/name=zookeeper-operator -n zookeeper-operator
```

When it gets ready, you will see something like this:

```bash
pod/zookeeper-operator-7975d7d66b-nh2tw condition met
```

If your _wait_ timedout, try another wait.

## Usage

#### Creating the simplest Zookeeper ensemble

Apply the following yaml to create the ensemble with 3 nodes.

```yaml
apiVersion: zookeeper.monime.sl/v1alpha1
kind: ZookeeperCluster
metadata:
  name: cluster-1
  namespace: zookeeper
spec:
  size: 3
  persistence:
    reclaimPolicy: "Delete"
```

#### Scale up the ensemble from 3 to 5 nodes:

Apply the following yaml to update the `cluster-1` ensemble.

```yaml
apiVersion: zookeeper.monime.sl/v1alpha1
kind: ZookeeperCluster
metadata:
  name: cluster-1
  namespace: zookeeper
spec:
  size: 5 # scale out
  persistence:
    reclaimPolicy: "Delete"
```