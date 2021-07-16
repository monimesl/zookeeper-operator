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
	"fmt"
	"github.com/monimesl/operator-helper/basetype"
	"github.com/monimesl/operator-helper/reconciler"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var (
	_ reconciler.Defaulting = &ZookeeperCluster{}
)

// +kubebuilder:object:root=true

// ZookeeperClusterList contains a list of ZookeeperCluster
type ZookeeperClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZookeeperCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZookeeperCluster{}, &ZookeeperClusterList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ZookeeperCluster is the Schema for the zookeeperclusters API
type ZookeeperCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZookeeperClusterSpec   `json:"spec,omitempty"`
	Status ZookeeperClusterStatus `json:"status,omitempty"`
}

func (in *ZookeeperCluster) CreateLabels(addPodLabels bool, more map[string]string) map[string]string {
	return in.Spec.CreateLabels(in.Name, addPodLabels, more)
}

func (in *ZookeeperCluster) nameHasZkIndicator() bool {
	return strings.Contains(in.Name, "zk") || strings.Contains(in.Name, "zookeeper")
}

func (in *ZookeeperCluster) generateName() string {
	if in.nameHasZkIndicator() {
		return in.Name
	}
	return fmt.Sprintf("%s-zk", in.GetName())
}

// ConfigMapName defines the name of the configmap object
func (in *ZookeeperCluster) ConfigMapName() string {
	return in.generateName()
}

// StatefulSetName defines the name of the statefulset object
func (in *ZookeeperCluster) StatefulSetName() string {
	return in.generateName()
}

// ClientServiceName defines the name of the client service object
func (in *ZookeeperCluster) ClientServiceName() string {
	return in.generateName()
}

// HeadlessServiceName defines the name of the headless service object
func (in *ZookeeperCluster) HeadlessServiceName() string {
	return fmt.Sprintf("%s-headless", in.ClientServiceName())
}

// ClientServiceFQDN defines the FQDN of the client service object
func (in *ZookeeperCluster) ClientServiceFQDN() string {
	return fmt.Sprintf("%s.%s.svc.%s", in.ClientServiceName(), in.Namespace, in.Spec.ClusterDomain)
}

// HeadlessServiceFQDN defines the FQDN of the headless service object
func (in *ZookeeperCluster) HeadlessServiceFQDN() string {
	return fmt.Sprintf("%s.%s.svc.%s", in.HeadlessServiceName(), in.Namespace, in.Spec.ClusterDomain)
}

// IsSslClientSupported returns whether SSL client is supported
func (in *ZookeeperCluster) IsSslClientSupported() bool {
	return in.Spec.Ports.SecureClient > 0
}

// SetSpecDefaults set the defaults for the cluster spec and returns true otherwise false
func (in *ZookeeperCluster) SetSpecDefaults() bool {
	return in.Spec.setDefaults()
}

// SetStatusDefaults set the defaults for the cluster status and returns true otherwise false
func (in *ZookeeperCluster) SetStatusDefaults() bool {
	return in.Status.setDefaults()
}

// Image the bookkeeper docker image for the cluster
func (in *ZookeeperCluster) Image() basetype.Image {
	return basetype.Image{
		Tag:        in.Spec.ZookeeperVersion,
		Repository: imageRepository,
		PullPolicy: v1.PullIfNotPresent,
	}
}
