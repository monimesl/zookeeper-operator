package zookeepercluster

import (
	"github.com/skulup/operator-helper/reconciler"
	"github.com/skulup/zookeeper-operator/api/v1alpha1"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_              reconciler.Context    = &Reconciler{}
	_              reconciler.Reconciler = &Reconciler{}
	reconcileFuncs                       = []func(ctx reconciler.Context, cluster *v1alpha1.ZookeeperCluster) error{
		reconcileStatefulSet,
	}
)

type Reconciler struct {
	reconciler.Context
}

func (r *Reconciler) Configure(ctx reconciler.Context) error {
	r.Context = ctx
	return ctx.NewControllerBuilder().
		For(&v1alpha1.ZookeeperCluster{}).
		Owns(&v1beta1.PodDisruptionBudget{}).
		Owns(&v12.StatefulSet{}).
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Service{}).
		Complete(r)
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
