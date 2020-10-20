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
	v1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func ReconcileMetrics(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	return reconcileServiceMonitor(ctx, cluster)
}

func reconcileServiceMonitor(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.Metrics != nil {
		sm := &v1.ServiceMonitor{}
		return ctx.GetResource(types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		}, sm,
			nil,
			// Not Found
			func() error {
				sm = createServiceMonitor(cluster)
				ctx.Logger().Info("Creating the zookeeper serviceMonitor.",
					"ServiceMonitor.Name", sm.GetName(),
					"ServiceMonitor.Namespace", sm.GetNamespace())
				return ctx.Client().Create(context.TODO(), sm)
			},
		)
	}
	return nil
}

func createServiceMonitor(cluster *v1alpha1.ZookeeperCluster) *v1.ServiceMonitor {
	sm := cluster.Spec.Metrics.NewServiceMonitor(cluster.Name, cluster.Namespace, cluster.Spec.Labels,
		metav1.LabelSelector{MatchLabels: cluster.CreateLabels(false, nil)}, serviceMetricsPortName)
	sm.Spec.NamespaceSelector = v1.NamespaceSelector{MatchNames: []string{cluster.Namespace}}
	return sm
}
