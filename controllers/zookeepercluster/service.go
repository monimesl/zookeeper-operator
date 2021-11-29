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
	"github.com/monimesl/operator-helper/k8s/service"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReconcileServices reconcile the services of the specified cluster
func ReconcileServices(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
	if err = reconcileHeadlessService(ctx, cluster); err == nil {
		err = reconcileClientService(ctx, cluster)
	}
	return
}

func reconcileClientService(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	svc := &v1.Service{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.ClientServiceName(),
		Namespace: cluster.Namespace,
	}, svc,
		nil,
		// Not Found
		func() (err error) {
			svc = createClientService(cluster)
			if err = ctx.SetOwnershipReference(cluster, svc); err == nil {
				ctx.Logger().Info("Creating the zookeeper client service.",
					"Service.Name", svc.GetName(),
					"Service.Namespace", svc.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), svc); err == nil {
					ctx.Logger().Info("Service creation success.",
						"Service.Name", svc.GetName(),
						"Service.Namespace", svc.GetNamespace())
				}
			}
			return
		})
}

func reconcileHeadlessService(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	svc := &v1.Service{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.HeadlessServiceName(),
		Namespace: cluster.Namespace,
	}, svc,
		nil,
		// Not Found
		func() (err error) {
			svc = createHeadlessService(cluster)
			if err = ctx.SetOwnershipReference(cluster, svc); err == nil {
				ctx.Logger().Info("Creating the zookeeper headless service.",
					"Service.Name", svc.GetName(),
					"Service.Namespace", svc.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), svc); err == nil {
					ctx.Logger().Info("Service creation success.",
						"Service.Name", svc.GetName(),
						"Service.Namespace", svc.GetNamespace())
				}
			}
			return
		})
}

func createClientService(c *v1alpha1.ZookeeperCluster) *v1.Service {
	return createService(c, c.ClientServiceName(), true, servicePorts(c.Spec.Ports))
}

func createHeadlessService(c *v1alpha1.ZookeeperCluster) *v1.Service {
	return createService(c, c.HeadlessServiceName(), false, servicePorts(c.Spec.Ports))
}

func createService(c *v1alpha1.ZookeeperCluster, name string, hasClusterIP bool, servicePorts []v1.ServicePort) *v1.Service {
	labels := c.GenerateLabels()
	clusterIP := ""
	if !hasClusterIP {
		clusterIP = v1.ClusterIPNone
	}
	if c.IsSslClientSupported() {
		servicePorts = append(servicePorts,
			v1.ServicePort{
				Name: v1alpha1.SecureClientPortName,
				Port: c.Spec.Ports.SecureClient},
		)
	}
	srv := service.New(c.Namespace, name, labels, v1.ServiceSpec{
		ClusterIP: clusterIP,
		Selector:  labels,
		Ports:     servicePorts,
	})
	srv.Annotations = c.GenerateAnnotations()
	return srv
}

func servicePorts(ports *v1alpha1.Ports) []v1.ServicePort {
	return []v1.ServicePort{
		{Name: v1alpha1.AdminPortName, Port: ports.Admin},
		{Name: v1alpha1.ClientPortName, Port: ports.Client},
		{Name: v1alpha1.LeaderPortName, Port: ports.Leader},
		{Name: v1alpha1.QuorumPortName, Port: ports.Quorum},
		{Name: v1alpha1.ServiceMetricsPortName, Port: ports.Metrics},
	}
}
