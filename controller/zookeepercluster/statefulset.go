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
	"github.com/monimesl/operator-helper/k8s/pod"
	"github.com/monimesl/operator-helper/k8s/pvc"
	"github.com/monimesl/operator-helper/k8s/statefulset"
	"github.com/monimesl/operator-helper/reconciler"
	"github.com/monimesl/zookeeper-operator/api/v1alpha1"
	"github.com/monimesl/zookeeper-operator/internal/zk"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math"
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
				if err := zk.UpdateZkClusterMetadata(cluster); err != nil {
					return err
				}
				if err := reconcilePodDisruptionBudget(ctx, cluster); err != nil {
					return err
				}
				if err := updateStatefulset(ctx, sts, cluster); err != nil {
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
			if err := reconcilePodDisruptionBudget(ctx, cluster); err != nil {
				return err
			}
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
		{Name: PvcDataVolumeName, MountPath: c.Spec.Dirs.Data},
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

func createStartupProbe() *v12.Probe {
	return &v12.Probe{
		PeriodSeconds:    5,
		FailureThreshold: 30,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/probeStartup.sh"}},
		},
	}
}
func createReadinessProbe() *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
		Handler: v12.Handler{
			Exec: &v12.ExecAction{Command: []string{"/scripts/probeReadiness.sh"}},
		},
	}
}

func createLivenessProbe() *v12.Probe {
	return &v12.Probe{
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
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
			c.Spec.PersistenceVolume.ClaimSpec),
	}
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
	if cluster.Spec.Size < 3 {
		// For less than 3 nodes, we tolerate no node failure
		return intstr.FromInt(0)
	}
	// In zookeeper, if you can tolerate a node failure count of `F`
	// then you need `2F+1` nodes to form a quorum of healthy nodes.
	// i.f N = 2F + 1 => F = (N-1) / 2. Practically F = floor((N-1) / 2)
	i := int(math.Floor(float64(cluster.Spec.Size-1) / 2.0))
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
				MatchLabels: cluster.CreateLabels(true, nil),
			},
		},
	}
}
