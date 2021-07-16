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

package zookeepercluster

import (
	"context"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"github.com/monimesl/zookeeper-operator/internal/zk"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReconcileClusterStatus reconcile the status of the specified cluster
func ReconcileClusterStatus(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	err = createZkSizeNode(ctx, cluster)
	return err
}

func createZkSizeNode(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.Size != cluster.Status.Metadata.Size {
		sts := &v1.StatefulSet{}
		return ctx.GetResource(types.NamespacedName{
			Name:      cluster.StatefulSetName(),
			Namespace: cluster.Namespace,
		}, sts,
			func() (err error) {
				if err = zk.UpdateZkClusterMetadata(cluster); err == nil {
					cluster.Status.Metadata.Size = cluster.Spec.Size
					err = ctx.Client().Status().Update(context.TODO(), cluster)
				}
				return
			}, nil)
	}
	return nil
}
