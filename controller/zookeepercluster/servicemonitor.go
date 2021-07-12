/*
 * Copyright 2020 - now, the original author or authors.
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
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReconcileServiceMonitor reconcile the serviceMonitor of the specified cluster
func ReconcileServiceMonitor(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	return createServiceMonitor(ctx, cluster)
}

func createServiceMonitor(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.Metrics != nil {
		sm := &v12.ServiceMonitor{}
		return ctx.GetResource(types.NamespacedName{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		}, sm,
			func() (err error) {
				if *cluster.Status.Metadata.ServiceMonitorVersion != sm.ResourceVersion {
					err = updateStatusResourceVersion(ctx, cluster, sm)
				}
				return
			},
			// Not Found
			func() (err error) {
				sm = create(cluster)
				if err := ctx.SetOwnershipReference(cluster, sm); err != nil {
					return err
				}
				ctx.Logger().Info("Creating the zookeeper serviceMonitor.",
					"ServiceMonitor.Name", sm.GetName(),
					"ServiceMonitor.Namespace", sm.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), sm); err == nil {
					err = updateStatusResourceVersion(ctx, cluster, sm)
				}
				return
			},
		)
	}
	return nil
}

func updateStatusResourceVersion(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster, sm *v12.ServiceMonitor) error {
	cluster.Status.Metadata.ServiceMonitorVersion = &sm.ResourceVersion
	return ctx.Client().Update(context.TODO(), cluster)
}

func create(cluster *v1alpha1.ZookeeperCluster) *v12.ServiceMonitor {
	sm := cluster.Spec.Metrics.NewServiceMonitor(cluster.Name, cluster.Namespace, cluster.Spec.Labels,
		metav1.LabelSelector{MatchLabels: cluster.CreateLabels(false, nil)}, serviceMetricsPortName)
	sm.Spec.NamespaceSelector = v12.NamespaceSelector{MatchNames: []string{cluster.Namespace}}
	return sm
}
