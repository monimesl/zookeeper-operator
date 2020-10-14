package zookeepercluster

import (
	"context"
	"github.com/skulup/operator-helper/k8s/pods"
	"github.com/skulup/operator-helper/k8s/pvcs"
	"github.com/skulup/operator-helper/k8s/statefulset"
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	configVolume = "config"
	dataVolume   = "data"
)

func reconcileStatefulSet(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	sts := &v1.StatefulSet{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.StatefulSetName(),
		Namespace: cluster.Namespace,
	}, sts,
		// Found
		func() (err error) { return },
		// Not Found
		func() (err error) {
			sts = createStatefulSet(cluster)
			if err = ctx.SetOwnershipReference(cluster, sts); err == nil {
				ctx.Logger().Info("Creating the zookeeper statefulset.",
					"StatefulSet.Name", sts.GetName(),
					"StatefulSet.Namespace", sts.GetNamespace())
				if err = ctx.Client().Create(context.TODO(), sts); err == nil {
					ctx.Logger().Info("StatefulSet creation success.",
						"StatefulSet.Name", sts.GetName(),
						"StatefulSet.Namespace", sts.GetNamespace())
				}
			}
			return
		})
}

func createStatefulSet(c *v1alpha1.ZookeeperCluster) *v1.StatefulSet {
	podLabels := c.CreateLabels(true, nil)
	podTemplate := createPodTemplateSpec(c, podLabels)
	sPvcs := createPersistentVolumeClaims(c)
	sSpec := statefulset.NewSpec(1, c.HeadlessServiceName(), podLabels, sPvcs, podTemplate)
	s := statefulset.New(c.Namespace, c.StatefulSetName(), c.Spec.Labels, sSpec)
	s.Annotations = c.Spec.Annotations
	return s
}

func createPodTemplateSpec(c *v1alpha1.ZookeeperCluster, labels map[string]string) v12.PodTemplateSpec {
	return v12.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: c.StatefulSetName(),
			Labels:       labels,
		},
		Spec: createPodSpec(c),
	}
}

func createPodSpec(c *v1alpha1.ZookeeperCluster) v12.PodSpec {
	containerPorts := []v12.ContainerPort{
		{Name: "client-port", ContainerPort: c.Spec.Ports.Client},
		{Name: "metrics-port", ContainerPort: c.Spec.Ports.Metrics},
		{Name: "quorum-port", ContainerPort: c.Spec.Ports.Quorum},
		{Name: "leader-election-port", ContainerPort: c.Spec.Ports.Leader},
	}
	if c.IsSslClientSupported() {
		containerPorts = append(containerPorts, v12.ContainerPort{
			Name:          "secure-client-port",
			ContainerPort: c.Spec.Ports.SecureClient,
		})
	}
	volumeMounts := []v12.VolumeMount{
		{Name: configVolume, MountPath: "/config"},
		{Name: dataVolume, MountPath: c.Spec.Dirs.Data},
	}
	if c.Spec.Dirs.Log != "" {
		volumeMounts = append(volumeMounts, v12.VolumeMount{Name: "log", MountPath: c.Spec.Dirs.Log})
	}
	container := v12.Container{
		Name:            "zookeeper",
		Ports:           containerPorts,
		Image:           c.Spec.Image.ToString(),
		ImagePullPolicy: c.Spec.Image.PullPolicy,
		VolumeMounts:    volumeMounts,
		Env:             pods.DecorateContainerEnvVars(true, c.Spec.Env...),
	}
	volumes := []v12.Volume{
		{
			Name: configVolume,
			VolumeSource: v12.VolumeSource{
				ConfigMap: &v12.ConfigMapVolumeSource{
					LocalObjectReference: v12.LocalObjectReference{
						Name: c.ConfigMapName(),
					},
				},
			},
		},
	}
	return pods.NewPodSpec(c.Spec.PodConfig, volumes, nil, []v12.Container{container})
}

func createPersistentVolumeClaims(c *v1alpha1.ZookeeperCluster) []v12.PersistentVolumeClaim {
	return []v12.PersistentVolumeClaim{
		pvcs.New(c.Namespace, dataVolume,
			c.CreateLabels(false, nil),
			c.Spec.PersistenceVolume.ClaimSpec),
	}
}
