package zookeepercluster

import (
	"context"
	"github.com/skulup/operator-helper/k8s/service"
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func reconcileServices(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) (err error) {
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
		// Found
		func() (err error) { return },
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
		// Found
		func() (err error) { return },
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
	return createService(c, c.ClientServiceName(), true,
		c.CreateLabels(false, nil), servicePorts(c.Spec.Ports))
}

func createHeadlessService(c *v1alpha1.ZookeeperCluster) *v1.Service {
	return createService(c, c.HeadlessServiceName(), false,
		c.CreateLabels(false, nil), servicePorts(c.Spec.Ports))
}

func createService(c *v1alpha1.ZookeeperCluster, name string, hasClusterIp bool,
	labels map[string]string, servicePorts []v1.ServicePort) *v1.Service {
	clusterIp := ""
	if !hasClusterIp {
		clusterIp = v1.ClusterIPNone
	}
	if c.IsSslClientSupported() {
		servicePorts = append(servicePorts,
			v1.ServicePort{
				Name: "secure-client-port",
				Port: c.Spec.Ports.SecureClient},
		)
	}
	return service.New(c.Namespace, name, labels, v1.ServiceSpec{
		ClusterIP: clusterIp,
		Selector:  labels,
		Ports:     servicePorts,
	})
}

func servicePorts(ports *v1alpha1.Ports) []v1.ServicePort {
	return []v1.ServicePort{
		{Name: "admin-port", Port: ports.Admin},
		{Name: "client-port", Port: ports.Client},
		{Name: "metrics-port", Port: ports.Metrics},
		{Name: "leader-port", Port: ports.Leader},
		{Name: "quorum-port", Port: ports.Quorum},
	}
}
