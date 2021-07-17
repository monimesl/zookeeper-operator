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
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/k8s/pvc"
	"github.com/monimesl/operator-helper/k8s/statefulset"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"github.com/monimesl/zookeeper-operator/internal/zk"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	configVolume = "config"
	// PvcDataVolumeName defines the name if the nodes PVCs
	PvcDataVolumeName = "data"
)

var (
	defaultTerminationGracePeriod int64 = 600
)

// ReconcileStatefulSet reconcile the statefulset of the specified cluster
func ReconcileStatefulSet(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	sts := &v1.StatefulSet{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.StatefulSetName(),
		Namespace: cluster.Namespace,
	}, sts,
		// Found
		func() error {
			if cluster.Spec.Size != *sts.Spec.Replicas {
				if err := zk.UpdateMetadata(cluster); err != nil {
					return err
				}
				if err := updateStatefulset(ctx, sts, cluster); err != nil {
					return err
				}
				if err := updateStatefulsetPvs(ctx, sts, cluster); err != nil {
					return err
				}
			}
			return nil
		},
		// Not Found
		func() error {
			sts = createStatefulSet(cluster)
			if err := ctx.SetOwnershipReference(cluster, sts); err != nil {
				return err
			}
			ctx.Logger().Info("Creating the zookeeper statefulset.",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace())
			if err := ctx.Client().Create(context.TODO(), sts); err != nil {
				return err
			}
			ctx.Logger().Info("StatefulSet creation success.",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace())
			return nil
		})
}

func updateStatefulset(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.ZookeeperCluster) error {
	sts.Spec.Replicas = &cluster.Spec.Size
	ctx.Logger().Info("Updating the zookeeper statefulset.",
		"StatefulSet.Name", sts.GetName(),
		"StatefulSet.Namespace", sts.GetNamespace(), "NewReplicas", cluster.Spec.Size)
	return ctx.Client().Update(context.TODO(), sts)
}

func updateStatefulsetPvs(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.ZookeeperCluster) error {
	if cluster.Spec.Persistence.ReclaimPolicy != v1alpha1.VolumeReclaimPolicyDelete && cluster.Spec.Metrics == nil {
		// Keep the orphan PVC since the reclaimed policy said so
		return nil
	}
	pvcList, err := pvc.ListAllWithMatchingLabels(ctx.Client(), sts.Namespace, sts.Spec.Template.Labels)
	if err != nil {
		return err
	}
	for _, item := range pvcList.Items {
		if oputil.IsOrdinalObjectIdle(item.Name, int(*sts.Spec.Replicas)) {
			toDel := &v12.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      item.Name,
					Namespace: item.Namespace,
				},
			}
			ctx.Logger().Info("Deleting the idle pvc. ",
				"StatefulSet.Name", sts.GetName(),
				"StatefulSet.Namespace", sts.GetNamespace(),
				"PVC.Namespace", toDel.GetNamespace(), "PVC.Name", toDel.GetName())
			err = ctx.Client().Delete(context.TODO(), toDel)
			if err != nil {
				return fmt.Errorf("error on deleing the pvc (%s): %v", toDel.Name, err)
			}
		}
	}
	return nil
}

func createStatefulSet(c *v1alpha1.ZookeeperCluster) *v1.StatefulSet {
	pvcs := createPersistentVolumeClaims(c)
	labels := c.CreateLabels(true, nil)
	templateSpec := createPodTemplateSpec(c, labels)
	spec := statefulset.NewSpec(c.Spec.Size, c.HeadlessServiceName(), labels, pvcs, templateSpec)
	s := statefulset.New(c.Namespace, c.StatefulSetName(), labels, spec)
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
		{Name: serviceMetricsPortName, ContainerPort: c.Spec.Ports.Metrics},
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
		{Name: PvcDataVolumeName, MountPath: c.Spec.Directories.Data},
	}
	if c.Spec.Directories.Log != "" {
		volumeMounts = append(volumeMounts, v12.VolumeMount{Name: "log", MountPath: c.Spec.Directories.Log})
	}
	image := c.Image()
	container := v12.Container{
		Name:            "zookeeper",
		Ports:           containerPorts,
		Image:           image.ToString(),
		ImagePullPolicy: image.PullPolicy,
		VolumeMounts:    volumeMounts,
		StartupProbe:    createStartupProbe(c.Spec.Probes.Startup),
		LivenessProbe:   createLivenessProbe(c.Spec.Probes.Liveness),
		ReadinessProbe:  createReadinessProbe(c.Spec.Probes.Readiness),
		Lifecycle:       &v12.Lifecycle{PreStop: createPreStopHandler()},
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.Env...),
		Command:         []string{"/scripts/start.sh"},
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

func createStartupProbe(probe *pod.Probe) *v12.Probe {
	return &v12.Probe{
		PeriodSeconds:    probe.PeriodSeconds,
		FailureThreshold: probe.FailureThreshold,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/probeStartup.sh"}},
		},
	}
}
func createReadinessProbe(probe *pod.Probe) *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/probeReadiness.sh"}},
		},
	}
}

func createLivenessProbe(probe *pod.Probe) *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/probeLiveness.sh"}},
		},
	}
}

func createPreStopHandler() *v12.Handler {
	return &v12.Handler{Exec: &v12.ExecAction{Command: []string{"/scripts/stop.sh"}}}
}

func createPersistentVolumeClaims(c *v1alpha1.ZookeeperCluster) []v12.PersistentVolumeClaim {
	return []v12.PersistentVolumeClaim{
		pvc.New(c.Namespace, PvcDataVolumeName,
			c.CreateLabels(false, nil),
			c.Spec.Persistence.ClaimSpec),
	}
}
