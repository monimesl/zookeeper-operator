[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/skulup/zookeeper-operator)](https://goreportcard.com/report/github.com/skulup/zookeeper-operator)
![Build](https://github.com/skulup/zookeeper-operator/workflows/Build/badge.svg)

# Apache Zookeeper Operator

**Project Status: *alpha***

Only core features has been implemented; 
On fixing bug and implementing other features, the API spec, status may change, 
but we'll try to keep them as backward compatible as possible

## Overview
The Zookeeper Operator enable native [Kubernetes](https://kubernetes.io/) deployment and management of Apache Zookeeper Ensemble.
To set up the cluster, the operator uses [Zookeeper Dynamic Configuration](https://zookeeper.apache.org/doc/current/zookeeperReconfig.html) which is supported by version __3.6+__ .
For now, version __3.6.2__ is used as the installed version.

## Prerequisites
The operator needs a kubernetes cluster with a version __>= v1.15.0__ . 
If you're using Helm to install the operator, your need version __>= 3.0.0__ .

## Operator Installation
The operator can be installed and upgrade by using our [helm chart](https://github.com/skulup/zookeeper-operator/tree/master/deployments/charts)
or directly using the [manifest file](https://github.com/skulup/zookeeper-operator/blob/master/deployments/operator-manifest.yaml).
We however do recommend using the [helm chart](https://github.com/skulup/zookeeper-operator/tree/master/deployments/charts).

### Via [Manifest file](https://github.com/skulup/zookeeper-operator/blob/master/deployments/operator-manifest.yaml)
If you don't have [Helm](https://helm.sh/) or its required version, or you just want to try the operator quickly, this option is then ideal. 
We provide a manifest file per operator version. The below command will install the operator of version __v0.1.0__.
You can use the manifest of the master branch to install the latest version.

Install the version __v0.1.0__:
```bash
 kubectl apply -f https://raw.githubusercontent.com/skulup/zookeeper-operator/v0.1.0/deployments/operator-manifest.yaml 
```

__OR__ install the latest version:
```bash
 kubectl apply -f https://raw.githubusercontent.com/skulup/zookeeper-operator/master/deployments/operator-manifest.yaml
```
Mind you, either command above will install a [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
and create a [ClusterRole](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/);
so the user issuing this command must have cluster-admin privileges.

### Via [Helm](https://helm.sh/)
First you need to add the chart's [repository](https://skulup.github.io/charts/) to our repo list:

```bash
helm repo add skulup https://skulup.github.io/charts
helm repo update
```
Create the operator namespace; we're doing this because Helm 3 no longer automatically create namespace.
```bash
kubectl create namespace zookeeper-operator
```

Now install the chart in the created namespace
```bash
helm install zookeeper-operator skulup/zookeeper-operator -n zookeeper-operator
```

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
####....