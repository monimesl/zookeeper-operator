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
	"github.com/skulup/operator-helper/k8s/pod"
	"github.com/skulup/operator-helper/k8s/pvc"
	"github.com/skulup/operator-helper/k8s/statefulset"
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	"github.com/skulup/zookeeper-operator/internal/zk_util"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	configVolume = "config"
	dataVolume   = "data"
)

var (
	defaultTerminationGracePeriod int64 = 600
)

func ReconcileStatefulSet(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	sts := &v1.StatefulSet{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.StatefulSetName(),
		Namespace: cluster.Namespace,
	}, sts,
		// Found
		func() (err error) {
			if cluster.Spec.Size != *sts.Spec.Replicas {
				if err = zk_util.UpdateZkClusterMetaSize(cluster); err == nil {
					err = updateStatefulset(ctx, sts, cluster)
				}
			}
			return
		},
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

func updateStatefulset(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.ZookeeperCluster) error {
	sts.Spec.Replicas = &cluster.Spec.Size
	ctx.Logger().Info("Updating the zookeeper statefulset.",
		"StatefulSet.Name", sts.GetName(),
		"StatefulSet.Namespace", sts.GetNamespace(), "NewReplicas", cluster.Spec.Size)
	return ctx.Client().Update(context.TODO(), sts)
}

func createStatefulSet(c *v1alpha1.ZookeeperCluster) *v1.StatefulSet {
	pvcs := createPersistentVolumeClaims(c)
	podLabels := c.CreateLabels(true, nil)
	templateSpec := createPodTemplateSpec(c, podLabels)
	spec := statefulset.NewSpec(c.Spec.Size, c.HeadlessServiceName(), podLabels, pvcs, templateSpec)
	s := statefulset.New(c.Namespace, c.StatefulSetName(), c.Spec.Labels, spec)
	s.Annotations = c.Spec.Annotations
	return s
}

func createPodTemplateSpec(c *v1alpha1.ZookeeperCluster, labels map[string]string) v12.PodTemplateSpec {
	return pod.NewTemplateSpec("", c.StatefulSetName(), labels, nil, createPodSpec(c))
}

func createPodSpec(c *v1alpha1.ZookeeperCluster) v12.PodSpec {
	containerPorts := []v12.ContainerPort{
		{Name: "admin-port", ContainerPort: c.Spec.Ports.Admin},
		{Name: "client-port", ContainerPort: c.Spec.Ports.Client},
		{Name: "metrics-port", ContainerPort: c.Spec.Ports.Metrics},
		{Name: "quorum-port", ContainerPort: c.Spec.Ports.Quorum},
		{Name: "leader-port", ContainerPort: c.Spec.Ports.Leader},
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
		StartupProbe:    createStartupProbe(),
		LivenessProbe:   createLivenessProbe(),
		ReadinessProbe:  createReadinessProbe(),
		Lifecycle:       &v12.Lifecycle{PreStop: createPreStopHandler()},
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.Env...),
		Command:         []string{"/scripts/zkStart.sh"},
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
	spec := pod.NewSpec(c.Spec.PodConfig, volumes, nil, []v12.Container{container})
	spec.TerminationGracePeriodSeconds = &defaultTerminationGracePeriod
	return spec
}

func createStartupProbe() *v12.Probe {
	return &v12.Probe{
		PeriodSeconds:    5,
		FailureThreshold: 30,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/zkProbeStartup.sh"}},
		},
	}
}
func createReadinessProbe() *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/zkProbeReadiness.sh"}},
		},
	}
}

func createLivenessProbe() *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/zkProbeLiveness.sh"}},
		},
	}
}

func createPreStopHandler() *v12.Handler {
	return &v12.Handler{Exec: &v12.ExecAction{Command: []string{"/scripts/zkStop.sh"}}}
}

func createPersistentVolumeClaims(c *v1alpha1.ZookeeperCluster) []v12.PersistentVolumeClaim {
	return []v12.PersistentVolumeClaim{
		pvc.New(c.Namespace, dataVolume,
			c.CreateLabels(false, nil),
			c.Spec.PersistenceVolume.ClaimSpec),
	}
}
