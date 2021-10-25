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
	defaultImageTag = "3.6.3"
)

const (
	defaultDataDir = "/data"
)

const (
	AdminPortName          = "admin-port"
	ClientPortName         = "client-port"
	LeaderPortName         = "leader-port"
	QuorumPortName         = "quorum-port"
	ServiceMetricsPortName = "metrics-port"
	ServiceMetricsPath     = "/metrics"
	SecureClientPortName   = "secure-client-port"
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
	// VolumeReclaimPolicyDelete deletes the volume after the cluster is deleted
	VolumeReclaimPolicyDelete = "Delete"
	// VolumeReclaimPolicyRetain retains the volume after the cluster is deleted
	VolumeReclaimPolicyRetain = "Retain"
)

const (
	defaultStorageVolumeSize = "10Gi"
	defaultClusterDomain     = "cluster.local"
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

	// Log4jProps defines the log4j.properties data
	Log4jProps string `json:"log4jProps,omitempty"`

	// Log4jQuietProps defines the log4j-quiet.properties data
	Log4jQuietProps string `json:"log4jQuietProps,omitempty"`

	// Persistence configures your node storage
	// +optional
	Persistence *Persistence `json:"persistence,omitempty"`

	// PodConfig defines common configuration for the zookeeper pods
	PodConfig basetype.PodConfig `json:"podConfig,omitempty"`
	// ProbeConfig defines the probing settings for the zookeeper containers
	ProbeConfig      *pod.Probes      `json:"probeConfig,omitempty"`
	MonitoringConfig MonitoringConfig `json:"monitoringConfig,omitempty"`

	// Env defines environment variables for the zookeeper statefulset pods
	Env []v1.EnvVar `json:"env,omitempty"`

	// Labels defines the labels to attach to the zookeeper statefulset pods
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations defines the annotations to attach to the zookeeper statefulset and services
	Annotations map[string]string `json:"annotations,omitempty"`

	// ClusterDomain defines the cluster domain for the cluster
	// It defaults to cluster.local
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

type MonitoringConfig struct {
	// Enabled defines whether this monitoring is enabled or not.
	Enabled bool `json:"enabled,omitempty"`
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
	// If it's set to Delete and the zookeeper cluster is deleted, the corresponding PVCs will be deleted.
	// The default value is Retain.
	// +kubebuilder:validation:Enum="Delete";"Retain"
	ReclaimPolicy VolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`
	// ClaimSpec describes the common attributes of storage devices
	// and allows a Source for provider-specific attributes
	ClaimSpec v1.PersistentVolumeClaimSpec `json:"claimSpec,omitempty"`
}

func (in *Persistence) setDefault() (changed bool) {
	if in.ReclaimPolicy != VolumeReclaimPolicyDelete && in.ReclaimPolicy != VolumeReclaimPolicyRetain {
		in.ReclaimPolicy = VolumeReclaimPolicyRetain
		changed = true
	}
	storage, ok := in.ClaimSpec.Resources.Requests[v1.ResourceStorage]
	if !ok || storage.IsZero() {
		changed = true
		if in.ClaimSpec.Resources.Requests == nil {
			in.ClaimSpec.Resources.Requests = map[v1.ResourceName]resource.Quantity{}
		}
		in.ClaimSpec.Resources.Requests[v1.ResourceStorage] = resource.MustParse(defaultStorageVolumeSize)
	}
	in.ClaimSpec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	return
}

func (in *ZookeeperClusterSpec) setMetricsDefault() (changed bool) {
	return false
}

func (in *ZookeeperClusterSpec) CreateLabels(clusterName string, addPodLabels bool, more map[string]string) map[string]string {
	labels := in.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	if addPodLabels {
		for k, v := range in.PodConfig.Labels {
			labels[k] = v
		}
	}
	for k, v := range more {
		labels[k] = v
	}
	labels[k8s.LabelAppName] = "zookeeper"
	labels[k8s.LabelAppInstance] = clusterName
	labels[k8s.LabelAppVersion] = in.ZookeeperVersion
	labels[k8s.LabelAppManagedBy] = internal.OperatorName
	return labels
}

// setDefaults set the defaults for the cluster spec and returns true otherwise false
func (in *ZookeeperClusterSpec) setDefaults() (changed bool) {
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
	if in.PodConfig.TerminationGracePeriodSeconds == nil {
		changed = true
		in.PodConfig.TerminationGracePeriodSeconds = &defaultTerminationGracePeriod
	}
	return
}
