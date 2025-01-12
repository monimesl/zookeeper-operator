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
	"github.com/monimesl/operator-helper/k8s"
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/k8s/pvc"
	"github.com/monimesl/operator-helper/oputil"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

const (
	configVolume         = "config"
	PvcDataVolumeName    = "data"
	PvcDataLogVolumeName = "data-log"
)

// ReconcileStatefulSet reconcile the statefulset of the specified cluster
func ReconcileStatefulSet(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error {
	sts := &v1.StatefulSet{}
	return ctx.GetResource(types.NamespacedName{
		Name:      cluster.GetName(),
		Namespace: cluster.Namespace,
	}, sts,
		// Found
		func() error {
			if shouldUpdateStatefulSet(ctx, cluster, sts) {
				if err := updateStatefulset(ctx, sts, cluster); err != nil {
					return err
				}
				if err := updateStatefulsetPVCs(ctx, sts); err != nil {
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

func shouldUpdateStatefulSet(ctx reconciler.Context, c *v1alpha1.ZookeeperCluster, sts *v1.StatefulSet) bool {
	if *c.Spec.Size != *sts.Spec.Replicas {
		ctx.Logger().Info("Zookeeper cluster size changed",
			"from", *sts.Spec.Replicas, "to", *c.Spec.Size)
		return true
	}
	if c.Spec.ZookeeperVersion != c.Status.Metadata.ZkVersion {
		ctx.Logger().Info("Zookeeper version changed",
			"from", c.Status.Metadata.ZkVersion, "to", c.Spec.ZookeeperVersion,
		)
		return true
	}
	if c.Spec.ZkConfig != c.Status.Metadata.ZkConfig {
		ctx.Logger().Info("Zookeeper cluster config changed",
			"from", c.Status.Metadata.ZkConfig, "to", c.Spec.ZkConfig,
		)
		return true
	}
	return false
}

func updateStatefulset(ctx reconciler.Context, sts *v1.StatefulSet, cluster *v1alpha1.ZookeeperCluster) error {
	sts.Spec.Replicas = cluster.Spec.Size
	containers := sts.Spec.Template.Spec.Containers
	for i, container := range containers {
		if container.Name == "zookeeper" {
			container.Image = cluster.Image().ToString()
			containers[i] = container
		}
	}
	sts.Spec.Template.Spec.Containers = containers
	ctx.Logger().Info("Updating the zookeeper statefulset.",
		"StatefulSet.Name", sts.GetName(),
		"StatefulSet.Namespace", sts.GetNamespace(),
		"NewReplicas", cluster.Spec.Size,
		"NewVersion", cluster.Spec.ZookeeperVersion)
	return ctx.Client().Update(context.TODO(), sts)
}

func updateStatefulsetPVCs(ctx reconciler.Context, sts *v1.StatefulSet) error {
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
				return fmt.Errorf("error on deleing the pvc (%s): %w", toDel.Name, err)
			}
		}
	}
	return nil
}

func createStatefulSet(c *v1alpha1.ZookeeperCluster) *v1.StatefulSet {
	return &v1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetName(),
			Namespace: c.Namespace,
			Labels: mergeLabels(c.GenerateLabels(), map[string]string{
				k8s.LabelAppVersion: c.Spec.ZookeeperVersion,
				"version":           c.Spec.ZookeeperVersion,
			}),
			Annotations: c.GenerateAnnotations(),
		},
		Spec: v1.StatefulSetSpec{
			ServiceName: c.HeadlessServiceName(),
			Replicas:    c.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: c.GenerateLabels(),
			},
			UpdateStrategy: v1.StatefulSetUpdateStrategy{
				Type: v1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: v1.OrderedReadyPodManagement,
			Template: v12.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: c.GetName(),
					Labels: mergeLabels(
						c.GenerateLabels(),
						c.Spec.PodConfig.Labels,
					),
					Annotations: c.Spec.PodConfig.Annotations,
				},
				Spec: createPodSpec(c),
			},
			VolumeClaimTemplates: createPersistentVolumeClaims(c),
		},
	}
}

func createPodSpec(c *v1alpha1.ZookeeperCluster) v12.PodSpec {
	containerPorts := []v12.ContainerPort{
		{Name: v1alpha1.AdminPortName, ContainerPort: c.Spec.Ports.Admin},
		{Name: v1alpha1.ClientPortName, ContainerPort: c.Spec.Ports.Client},
		{Name: v1alpha1.QuorumPortName, ContainerPort: c.Spec.Ports.Quorum},
		{Name: v1alpha1.LeaderPortName, ContainerPort: c.Spec.Ports.Leader},
		{Name: v1alpha1.ServiceMetricsPortName, ContainerPort: c.Spec.Ports.Metrics},
	}
	if c.IsSslClientSupported() {
		containerPorts = append(containerPorts, v12.ContainerPort{
			Name:          v1alpha1.SecureClientPortName,
			ContainerPort: c.Spec.Ports.SecureClient,
		})
	}
	dataDir := c.Spec.Directories.Data
	dataDir = strings.TrimSuffix(dataDir, "/")
	volumeMounts := []v12.VolumeMount{
		{Name: configVolume, MountPath: "/config"},
		{Name: PvcDataVolumeName, MountPath: dataDir},
		{Name: PvcDataLogVolumeName, MountPath: dataDir + "-log"},
	}
	if c.Spec.Directories.Log != "" {
		volumeMounts = append(volumeMounts, v12.VolumeMount{Name: "log", MountPath: c.Spec.Directories.Log})
	}
	image := c.Image()
	container := v12.Container{
		Name:            "zookeeper",
		VolumeMounts:    volumeMounts,
		Ports:           containerPorts,
		Image:           image.ToString(),
		ImagePullPolicy: image.PullPolicy,
		Resources:       c.Spec.PodConfig.Spec.Resources,
		StartupProbe:    createStartupProbe(c.Spec.ProbeConfig.Startup),
		LivenessProbe:   createLivenessProbe(c.Spec.ProbeConfig.Liveness),
		ReadinessProbe:  createReadinessProbe(c.Spec.ProbeConfig.Readiness),
		Lifecycle:       &v12.Lifecycle{PreStop: createPreStopHandler()},
		Env:             pod.DecorateContainerEnvVars(true, c.Spec.PodConfig.Spec.Env...),
		Command:         []string{"/scripts/start.sh"},
	}
	volumes := []v12.Volume{
		{
			Name: configVolume,
			VolumeSource: v12.VolumeSource{
				ConfigMap: &v12.ConfigMapVolumeSource{
					LocalObjectReference: v12.LocalObjectReference{
						Name: c.GetName(),
					},
				},
			},
		},
	}
	return pod.NewSpec(c.Spec.PodConfig, volumes, nil, []v12.Container{container})
}

func createStartupProbe(probe *pod.Probe) *v12.Probe {
	return probe.ToK8sProbe(v12.ProbeHandler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeStartup.sh"}},
	})
}
func createReadinessProbe(probe *pod.Probe) *v12.Probe {
	return probe.ToK8sProbe(v12.ProbeHandler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeReadiness.sh"}},
	})
}

func createLivenessProbe(probe *pod.Probe) *v12.Probe {
	return probe.ToK8sProbe(v12.ProbeHandler{
		Exec: &v12.ExecAction{Command: []string{"/scripts/probeLiveness.sh"}},
	})
}

func createPreStopHandler() *v12.LifecycleHandler {
	return &v12.LifecycleHandler{Exec: &v12.ExecAction{Command: []string{"/scripts/stop.sh"}}}
}

func createPersistentVolumeClaims(c *v1alpha1.ZookeeperCluster) []v12.PersistentVolumeClaim {
	persistence := c.Spec.Persistence
	pvcs := []v12.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: PvcDataVolumeName,
				Labels: mergeLabels(
					c.Spec.Labels,
					map[string]string{
						"app": c.GetName(),
					},
				),
				Annotations: c.Spec.Persistence.Annotations,
			},
			Spec: persistence.VolumeClaimSpec,
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: PvcDataLogVolumeName,
				Labels: mergeLabels(
					c.Spec.Labels,
					map[string]string{
						"app": c.GetName(),
					},
				),
				Annotations: c.Spec.Persistence.Annotations,
			},
			Spec: createDataLogVolumeClaimSpec(persistence.VolumeClaimSpec, c.Spec.GetDefaultDataLogStorageVolumeSize()),
		},
	}
	return pvcs
}

func createDataLogVolumeClaimSpec(spec v12.PersistentVolumeClaimSpec, size string) v12.PersistentVolumeClaimSpec {
	return v12.PersistentVolumeClaimSpec{
		AccessModes: spec.AccessModes,
		Selector:    spec.Selector,
		Resources: v12.ResourceRequirements{
			Requests: v12.ResourceList{
				v12.ResourceStorage: resource.MustParse(size),
			},
		},
		VolumeName:       spec.VolumeName,
		StorageClassName: spec.StorageClassName,
		VolumeMode:       spec.VolumeMode,
		DataSource:       spec.DataSource,
		DataSourceRef:    spec.DataSourceRef,
	}
}
