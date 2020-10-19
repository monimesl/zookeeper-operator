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

package controllers

import (
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	"github.com/skulup/zookeeper-operator/controllers/zookeepercluster"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_              reconciler.Context    = &ZookeeperClusterReconciler{}
	_              reconciler.Reconciler = &ZookeeperClusterReconciler{}
	reconcileFuncs                       = []func(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error{
		zookeepercluster.ReconcileConfigMap,
		zookeepercluster.ReconcileServices,
		zookeepercluster.ReconcileStatefulSet,
		zookeepercluster.ReconcilePodDisruptionBudget,
		zookeepercluster.ReconcileClusterStatus,
	}
)

type ZookeeperClusterReconciler struct {
	reconciler.Context
}

func (r *ZookeeperClusterReconciler) Configure(ctx reconciler.Context) error {
	r.Context = ctx
	return ctx.NewControllerBuilder().
		For(&v1alpha1.ZookeeperCluster{}).
		Owns(&v1beta1.PodDisruptionBudget{}).
		Owns(&v12.StatefulSet{}).
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (r *ZookeeperClusterReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	cluster := &v1alpha1.ZookeeperCluster{}
	return r.Run(request, cluster, func() (err error) {
		for _, fun := range reconcileFuncs {
			if err = fun(r, cluster); err != nil {
				break
			}
		}
		return
	})
}
