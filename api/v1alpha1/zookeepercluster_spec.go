/*
 * Copyright 2021 - now, the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"github.com/monimesl/operator-helper/basetype"
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/internal"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	_ reconciler.Defaulting = &ZookeeperCluster{}
)

const (
	imageRepository = "monime/zookeeper"
	defaultImageTag = "3.8.4"
)

const (
	defaultDataDir = "/data"
)

const (
	// VolumeReclaimPolicyDelete deletes the volume after the cluster is deleted
	VolumeReclaimPolicyDelete = "Delete"
	// VolumeReclaimPolicyRetain retains the volume after the cluster is deleted
	VolumeReclaimPolicyRetain = "Retain"
)

const (
	AdminPortName          = "http-admin"
	ClientPortName         = "tcp-client"
	LeaderPortName         = "tcp-leader"
	QuorumPortName         = "tcp-quorum"
	ServiceMetricsPortName = "http-metrics"
	SecureClientPortName   = "tls-secure-client"
)

const (
	defaultAdminPort          = 8080
	defaultClientPort         = 2181
	defaultMetricsPort        = 7000
	defaultSecureClientPort   = -1
	defaultQuorumPort         = 2888
	defaultLeaderElectionPort = 3888
)

const (
	defaultDataStorageVolumeSize    = "8Gi"
	defaultDataLogStorageVolumeSize = "3Gi"
	defaultClusterDomain            = "cluster.local"
)

var (
	defaultClusterSize            int32 = 3
	defaultTerminationGracePeriod int64 = 120
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZookeeperClusterSpec defines the desired state of ZookeeperCluster
type ZookeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	Size *int32 `json:"size,omitempty"`

	Directories *Directories `json:"directories,omitempty"`

	Ports *Ports `json:"ports,omitempty"`

	// ZookeeperVersion defines the version of zookeeper to use
	ZookeeperVersion string `json:"zookeeperVersion,omitempty"`
	// ImagePullPolicy describes a policy for if/when to pull the image
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ZkConfig defines the zoo.cfg data
	ZkConfig string `json:"zkCfg,omitempty"`

	// Persistence configures your node storage
	// +optional
	Persistence *Persistence `json:"persistence,omitempty"`

	// PodConfig defines common configuration for the zookeeper pods
	PodConfig basetype.PodConfig `json:"podConfig,omitempty"`
	// ProbeConfig defines the probing settings for the zookeeper containers
	ProbeConfig *pod.Probes `json:"probeConfig,omitempty"`

	// Labels defines the labels to attach to the zookeeper statefulset pods
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations defines the annotations to attach to the zookeeper statefulset and services
	Annotations map[string]string `json:"annotations,omitempty"`

	// ClusterDomain defines the cluster domain for the cluster
	// It defaults to cluster.local
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type Ports struct {
	Client       int32 `json:"client,omitempty"`
	SecureClient int32 `json:"secureClient,omitempty"`
	Metrics      int32 `json:"metrics,omitempty"`
	Quorum       int32 `json:"quorum,omitempty"`
	Leader       int32 `json:"leader,omitempty"`
	Admin        int32 `json:"admin,omitempty"`
}

func (in *Ports) setDefaults() (changed bool) {
	if in.Admin == 0 {
		changed = true
		in.Admin = defaultAdminPort
	}
	if in.Client == 0 {
		changed = true
		in.Client = defaultClientPort
	}
	if in.Metrics == 0 {
		changed = true
		in.Metrics = defaultMetricsPort
	}
	if in.SecureClient == 0 {
		changed = true
		in.SecureClient = defaultSecureClientPort
	}
	if in.Quorum == 0 {
		changed = true
		in.Quorum = defaultQuorumPort
	}
	if in.Leader == 0 {
		changed = true
		in.Leader = defaultLeaderElectionPort
	}
	return
}

type Directories struct {
	Data string `json:"data,omitempty"`
	Log  string `json:"log,omitempty"`
}

// VolumeReclaimPolicy defines the possible volume reclaim policy: Delete or Retain
type VolumeReclaimPolicy string

// Persistence defines cluster node persistence volume is configured
type Persistence struct {
	// ReclaimPolicy decides the fate of the PVCs after the cluster is deleted.
	// If it's set to Delete and the bookkeeper cluster is deleted, the corresponding
	// PVCs will be deleted. The default value is Retain.
	// +kubebuilder:validation:Enum="Delete";"Retain"
	ReclaimPolicy VolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`
	// Annotations defines the annotations to attach to the pod
	Annotations map[string]string `json:"annotations,omitempty"`
	// VolumeClaimSpec describes the common attributes of storage devices
	// and allows a Source for provider-specific attributes
	VolumeClaimSpec v1.PersistentVolumeClaimSpec `json:"volumeClaimSpec,omitempty"`
}

func (in *Persistence) setDefault() (changed bool) {
	if in.ReclaimPolicy != VolumeReclaimPolicyDelete && in.ReclaimPolicy != VolumeReclaimPolicyRetain {
		in.ReclaimPolicy = VolumeReclaimPolicyDelete
		changed = true
	}
	storage, ok := in.VolumeClaimSpec.Resources.Requests[v1.ResourceStorage]
	if !ok || storage.IsZero() {
		changed = true
		if in.VolumeClaimSpec.Resources.Requests == nil {
			in.VolumeClaimSpec.Resources.Requests = map[v1.ResourceName]resource.Quantity{}
		}
		in.VolumeClaimSpec.Resources.Requests[v1.ResourceStorage] = resource.MustParse(defaultDataStorageVolumeSize)
	}
	in.VolumeClaimSpec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	return
}

func (in *ZookeeperClusterSpec) setMetricsDefault() (changed bool) {
	return false
}

func (in *ZookeeperClusterSpec) createAnnotations() map[string]string {
	return in.Annotations
}

func (in *ZookeeperClusterSpec) GetDefaultDataLogStorageVolumeSize() string {
	return defaultDataLogStorageVolumeSize
}

func (in *ZookeeperClusterSpec) createLabels(clusterName string) map[string]string {
	labels := in.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app"] = "zookeeper"
	labels[k8s.LabelAppName] = "zookeeper"
	labels[k8s.LabelAppInstance] = clusterName
	labels[k8s.LabelAppManagedBy] = internal.OperatorName
	return labels
}

// setDefaults set the defaults for the cluster spec and returns true otherwise false
//
//nolint:nakedret
func (in *ZookeeperClusterSpec) setDefaults() (changed bool) { //nolint:cyclop
	if in.ZookeeperVersion == "" {
		changed = true
		in.ZookeeperVersion = defaultImageTag
	}
	if in.ImagePullPolicy == "" {
		changed = true
		in.ImagePullPolicy = v1.PullIfNotPresent
	}
	if in.Size == nil {
		changed = true
		in.Size = &defaultClusterSize
	}
	if in.ClusterDomain == "" {
		changed = true
		in.ClusterDomain = defaultClusterDomain
	}
	if in.Directories == nil {
		changed = true
		in.Directories = &Directories{
			Data: defaultDataDir,
		}
	}
	if in.Ports == nil {
		in.Ports = &Ports{}
		in.Ports.setDefaults()
		changed = true
	} else if in.Ports.setDefaults() {
		changed = true
	}
	if in.ProbeConfig == nil {
		changed = true
		in.ProbeConfig = &pod.Probes{}
		in.ProbeConfig.SetDefault()
	} else if in.ProbeConfig.SetDefault() {
		changed = true
	}
	if in.Persistence == nil {
		in.Persistence = &Persistence{}
		in.Persistence.setDefault()
		changed = true
	} else if in.Persistence.setDefault() {
		changed = true
	}
	if in.setMetricsDefault() {
		changed = true
	}
	if in.PodConfig.Spec.TerminationGracePeriodSeconds == nil {
		changed = true
		in.PodConfig.Spec.TerminationGracePeriodSeconds = &defaultTerminationGracePeriod
	}
	return
}
