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

package zk

import (
	"fmt"
	"github.com/go-zookeeper/zk"
	"github.com/monimesl/operator-helper/config"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"strconv"
	"strings"
	"time"
)

const (
	clusterSizeKey = "SIZE"
	// ClusterMetadataParentZNode defines the znode to store metadata for the ZookeeperCluster objects
	ClusterMetadataParentZNode = "/zookeeper-operator-clusters"
	serverRemoveDateNode       = ClusterMetadataParentZNode + "/last-removal-time"
)

type Client struct {
	conn                 *zk.Conn
	requiredNodesCreated bool
}

// UpdateZkClusterMetadata update the metadata of the specified cluster
func UpdateZkClusterMetadata(cluster *v1alpha1.ZookeeperCluster) error {
	if cl, err := NewZkClient(cluster); err != nil {
		return err
	} else {
		defer cl.Close()
		return cl.updateClusterSizeMeta(cluster)
	}
}

//NewZkClient creates a new zookeeper client connected to the specified cluster
func NewZkClient(cluster *v1alpha1.ZookeeperCluster) (*Client, error) {
	port := cluster.Spec.Ports.SecureClient
	if cluster.Spec.Ports.Client > 0 {
		port = cluster.Spec.Ports.Client
	}
	address := fmt.Sprintf("%s:%d", cluster.ClientServiceFQDN(), port)
	c, _, err := zk.Connect([]string{address}, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

func (c *Client) updateClusterSizeMeta(cluster *v1alpha1.ZookeeperCluster) error {
	cNode := clusterNode(cluster)
	config.RequireRootLogger().Info("Setting the cluster-size metadata in zookeeper",
		"cluster", cluster.GetName(), "zkPath", cNode, "size", cluster.Spec.Size)
	data := []byte(fmt.Sprintf("%s=%d", clusterSizeKey, cluster.Spec.Size))
	if err := c.createRequiredNodes(); err != nil {
		return err
	}
	if exists, _, err := c.conn.Exists(cNode); err != nil {
		return err
	} else if exists {
		currentSize, sts, err := c.getClusterSize(cNode)
		if err != nil {
			return err
		}
		config.RequireRootLogger().
			Info("ZookeeperCluster Metadata",
				"cluster", cluster.GetName(),
				"current[SIZE]", currentSize, "spec[SIZE]", cluster.Spec.Size)
		if cluster.Spec.Size != currentSize {
			if _, err := c.conn.Set(cNode, data, sts.Version); err != nil {
				return err
			}
		}
		return nil
	}
	return c.createNode(cNode, data)

}

func clusterNode(cluster *v1alpha1.ZookeeperCluster) string {
	return fmt.Sprintf("%s/%s", ClusterMetadataParentZNode, cluster.GetName())
}

func (c *Client) createRequiredNodes() (err error) {
	if !c.requiredNodesCreated {
		if err = c.createNode(ClusterMetadataParentZNode, nil); err == nil {
			if err = c.createNode(serverRemoveDateNode, nil); err == nil {
				c.requiredNodesCreated = true
			}
		}
	}
	return
}

func (c *Client) createNode(path string, data []byte) (err error) {
	config.RequireRootLogger().
		Info("Creating the operator metadata node",
			"path", path, "data", string(data))
	if _, err = c.conn.Create(path, data, 0, zk.WorldACL(zk.PermAll)); err == zk.ErrNodeExists {
		return nil
	}
	return
}

func (c *Client) getClusterSize(clusterNode string) (int32, *zk.Stat, error) {
	if _, err := c.conn.Sync(clusterNode); err != nil {
		return 0, nil, err
	}
	data, sts, err := c.conn.Get(clusterNode)
	if err != nil {
		return 0, nil, err
	}
	sizeStr := strings.ReplaceAll(string(data), clusterSizeKey+"=", "")
	if size, err := strconv.Atoi(sizeStr); err != nil {
		return 0, nil, err
	} else {
		return int32(size), sts, nil
	}

}

// Close closes the zookeeper connection
func (c *Client) Close() {
	config.RequireRootLogger().Info("Closing the zookeeper client")
	c.conn.Close()
}
