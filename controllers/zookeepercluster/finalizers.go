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
	"fmt"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"github.com/monimesl/zookeeper-operator/internal/zk"
)

const (
	finalizerNamePrefix = "zookeepercluster.monime.sl-finalizer"
)

// ReconcileFinalizer reconcile the finalizer of the specified cluster
func ReconcileFinalizer(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	finalizerName := generateFinalizerName(cluster)
	if cluster.DeletionTimestamp.IsZero() { //nolint:nestif
		if !oputil.ContainsWithPrefix(cluster.Finalizers, finalizerNamePrefix) {
			ctx.Logger().Info("Adding the finalizer to the cluster",
				"cluster", cluster.Name, "finalizer", finalizerName)
			cluster.Finalizers = append(cluster.Finalizers, finalizerName)
			return ctx.Client().Update(context.TODO(), cluster)
		}
	} else if oputil.Contains(cluster.Finalizers, finalizerName) {
		if *cluster.Spec.Size > 0 {
			// Clean the metadata before downscaling the cluster
			// otherwise  there'll be no way to write since the
			// cluster itself is the metadata store
			if err := cleanUpMetadata(ctx, cluster); err != nil {
				return fmt.Errorf("BookkeeperCluster object (%s) zookeeper znodes cleanup error: %w", cluster.Name, err)
			}
			zero := int32(0)
			cluster.Spec.Size = &zero
			ctx.Logger().Info("Downscaling the cluster to zero to prepare delete",
				"cluster", cluster.Name) // this gives every pod a graceful shutdown
			if err := ctx.Client().Update(context.TODO(), cluster); err != nil {
				return fmt.Errorf("ZookkeeperCluster object (%s) update error: %w", cluster.Name, err)
			}
			return nil
		}
		if err := cluster.WaitClusterTermination(ctx.Client()); err != nil {
			return fmt.Errorf("error on waiting for the pods to terminate (%s): %w", cluster.Name, err)
		}
		ctx.Logger().Info("Finalizing the cluster",
			"cluster", cluster.Name,
			"finalizers", cluster.Finalizers,
			"finalizer", finalizerName)
		cluster.Finalizers = oputil.Remove(finalizerName, cluster.Finalizers)
		ctx.Logger().Info("Saving updated cluster finalizers",
			"cluster", cluster.Name, "finalizers", cluster.Finalizers)
		if err := ctx.Client().Update(context.TODO(), cluster); err != nil {
			return fmt.Errorf("ZookkeeperCluster object (%s) update error: %w", cluster.Name, err)
		}
		ctx.Logger().Info("Cluster finalizers update and cleanup success.",
			"cluster", cluster.GetName())
		return nil
	}
	return nil
}

func cleanUpMetadata(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	ctx.Logger().Info("Cleaning up the metadata for cluster", "cluster", cluster.Name)
	if err = zk.DeleteMetadata(cluster); err != nil {
		return fmt.Errorf("error on deleting the zookeeper znodes for the cluster (%s): %w", cluster.Name, err)
	}
	return nil
}

func generateFinalizerName(cluster *v1alpha1.ZookeeperCluster) string {
	return fmt.Sprintf("%s-%s", finalizerNamePrefix, cluster.Name)
}
