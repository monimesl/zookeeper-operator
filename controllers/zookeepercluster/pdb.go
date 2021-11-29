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
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math"
)

// ReconcilePodDisruptionBudget reconcile the poddisruptionbudget of the specified cluster
func ReconcilePodDisruptionBudget(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	return reconcilePodDisruptionBudget(ctx, cluster)
}

func reconcilePodDisruptionBudget(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	pdb := &v1beta1.PodDisruptionBudget{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, pdb,
		func() error {
			newMaxFailureNodes := calculateMaxAllowedFailureNodes(cluster)
			if newMaxFailureNodes.IntVal != pdb.Spec.MaxUnavailable.IntVal {
				pdb.Spec.MaxUnavailable.IntVal = newMaxFailureNodes.IntVal
				ctx.Logger().Info("Updating the zookeeper poddisruptionbudget for cluster",
					"cluster", cluster.Name,
					"PodDisruptionBudget.Name", pdb.GetName(),
					"PodDisruptionBudget.Namespace", pdb.GetNamespace(),
					"MaxUnavailable", pdb.Spec.MaxUnavailable.IntVal)
				return ctx.Client().Update(context.TODO(), pdb)
			}
			return nil
		},
		// Not Found
		func() error {
			pdb = createPodDisruptionBudget(cluster)
			if err := ctx.SetOwnershipReference(cluster, pdb); err != nil {
				return err
			}
			ctx.Logger().Info("Creating the zookeeper poddisruptionbudget for cluster",
				"cluster", cluster.Name,
				"PodDisruptionBudget.Name", pdb.GetName(),
				"PodDisruptionBudget.Namespace", pdb.GetNamespace(),
				"MaxUnavailable", pdb.Spec.MaxUnavailable.IntVal)
			return ctx.Client().Create(context.TODO(), pdb)
		},
	)
}

func calculateMaxAllowedFailureNodes(cluster *v1alpha1.ZookeeperCluster) intstr.IntOrString {
	if *cluster.Spec.Size < 3 {
		// For less than 3 nodes, we tolerate no node failure
		return intstr.FromInt(0)
	}
	// In zookeeper, if you can tolerate a node failure count of `F`
	// then you need `2F+1` nodes to form a quorum of healthy nodes.
	// i.f N = 2F + 1 => F = (N-1) / 2. Practically F = floor((N-1) / 2)
	i := int(math.Floor(float64(*cluster.Spec.Size-1) / 2.0))
	return intstr.FromInt(i)
}

func createPodDisruptionBudget(cluster *v1alpha1.ZookeeperCluster) *v1beta1.PodDisruptionBudget {
	newMaxFailureNodes := calculateMaxAllowedFailureNodes(cluster)
	return &v1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			MaxUnavailable: &newMaxFailureNodes,
			Selector: &metav1.LabelSelector{
				MatchLabels: cluster.GenerateLabels(),
			},
		},
	}
}
