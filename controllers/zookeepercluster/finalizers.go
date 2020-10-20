/*
 * Copyright 2020 Skulup Ltd, Open Collaborators
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	"github.com/skulup/zookeeper-operator/controllers/utils"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerName = "zookeeperclusters.zookeeper.skulup.com-finalizer"
)

// ReconcileFinalizer reconcile the finalizer of the specified cluster
func ReconcileFinalizer(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.PersistenceVolume.ReclaimPolicy != v1alpha1.VolumeReclaimPolicyDelete && cluster.Spec.Metrics == nil {
		return nil
	}
	if cluster.DeletionTimestamp.IsZero() {
		if !utils.Contains(finalizerName, cluster.Finalizers) {
			ctx.Logger().Info("Adding the finalizer to the cluster",
				"cluster", cluster.Name, "finalizer", finalizerName)
			cluster.Finalizers = append(cluster.Finalizers, finalizerName)
			return ctx.Client().Update(context.TODO(), cluster)
		}
	} else if utils.Contains(finalizerName, cluster.Finalizers) {
		if err := deleteAllPVCs(ctx, cluster); err != nil {
			return err
		}
		cluster.Finalizers = utils.Remove(finalizerName, cluster.Finalizers)
		return ctx.Client().Update(context.TODO(), cluster)
	}
	return nil
}

func deleteAllPVCs(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.PersistenceVolume.ReclaimPolicy != v1alpha1.VolumeReclaimPolicyDelete {
		return nil
	}
	pvCs, err := getPVCs(ctx, cluster)
	if err != nil {
		return err
	}
	for _, pvc := range pvCs.Items {
		if err := deletePVC(ctx, &pvc, cluster); err != nil {
			return err
		}
	}
	return nil
}

func deletePVC(ctx reconciler.Context, pvc *v1.PersistentVolumeClaim, cluster *v1alpha1.ZookeeperCluster) error {
	ctx.Logger().Info("Deleting the PVC for cluster", "cluster", cluster.Name, "pvc", pvc.Name)
	if err := ctx.Client().Delete(context.TODO(), pvc); err != nil {
		ctx.Logger().Info("Error deleting the PVC for cluster",
			"cluster", cluster.Name, "pvc", pvc.Name, "error",
			err.Error())
		return err
	}
	return nil
}

func getPVCs(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (*v1.PersistentVolumeClaimList, error) {
	ctx.Logger().Info("Finding the PVCs for cluster", "cluster", cluster.Name)
	pvcSelector, err := v12.LabelSelectorAsSelector(&v12.LabelSelector{
		MatchLabels: cluster.CreateLabels(false, nil),
	})
	if err != nil {
		return nil, err
	}
	pvCs := &v1.PersistentVolumeClaimList{}
	if err := ctx.Client().List(context.TODO(), pvCs, &client.ListOptions{
		LabelSelector: pvcSelector,
		Namespace:     cluster.Namespace,
	}); err != nil {
		return nil, err
	}
	return pvCs, nil
}
