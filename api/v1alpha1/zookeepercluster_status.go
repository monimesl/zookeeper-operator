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

// ZookeeperClusterStatus defines the observed state of ZookeeperCluster
type ZookeeperClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	Metadata Metadata `json:"metadata,omitempty"`
}

// Metadata defines the metadata status of the ZookeeperCluster
type Metadata struct {
	Size                  int32             `json:"size,omitempty"`
	ZkVersion             string            `json:"zkVersion,omitempty"`
	ZkConfig              string            `json:"zkConfig,omitempty"`
	ServiceMonitorVersion *string           `json:"serviceMonitorVersion,omitempty"`
	Data                  map[string]string `json:"data,omitempty"`
}

// setDefaults set the defaults for the cluster status and returns true otherwise false
func (in *ZookeeperClusterStatus) setDefaults() (changed bool) {
	if in.Metadata.Data == nil {
		in.Metadata = Metadata{
			Data: make(map[string]string),
		}
		changed = true
	}
	return
}
