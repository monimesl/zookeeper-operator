/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/skulup/operator-helper/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ reconciler.Defaulting = &ZookeeperCluster{}
)

const defaultRepository = "skulup/zookeeper"
const defaultTag = "latest"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZookeeperClusterSpec defines the desired state of ZookeeperCluster
type ZookeeperClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//// Image defines the container image to use.
	//Image types.Image `json:"image,omitempty"`
	//
	//// PodConfig defines common configuration for the zookeeper pods
	//PodConfig types.PodConfig `json:"pod,omitempty"`
	//
	//// Labels defines the labels to attach to the broker deployment
	//Labels map[string]string `json:"labels,omitempty"`
	//
	//LabelSelector metav1.LabelSelector `json:"selector,omitempty"`
	//
	//// Annotations defines the annotations to attach to the broker deployment
	//Annotations map[string]string `json:"annotations,omitempty"`
}

// ZookeeperClusterStatus defines the observed state of ZookeeperCluster
type ZookeeperClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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

func (in *ZookeeperCluster) SetSpecDefaults() (changed bool) {
	return
}

func (in *ZookeeperCluster) SetStatusDefaults() (changed bool) {
	return
}

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
