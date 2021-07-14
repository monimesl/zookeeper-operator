/*
Copyright 2021 - now, the original author or authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/monimesl/operator-helper/config"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (in *ZookeeperCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-zookeeper-monime-sl-v1alpha1-zookeepercluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=zookeeper.monime.sl,resources=zookeeperclusters,verbs=create;update,versions=v1alpha1,name=mzookeepercluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &ZookeeperCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ZookeeperCluster) Default() {
	config.RequireRootLogger().Info("default", "name", in.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-zookeeper-monime-sl-v1alpha1-zookeepercluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=zookeeper.monime.sl,resources=zookeeperclusters,verbs=create;update,versions=v1alpha1,name=vzookeepercluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &ZookeeperCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ZookeeperCluster) ValidateCreate() error {
	config.RequireRootLogger().Info("validate create", "name", in.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ZookeeperCluster) ValidateUpdate(old runtime.Object) error {
	config.RequireRootLogger().Info("validate update", "name", in.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ZookeeperCluster) ValidateDelete() error {
	config.RequireRootLogger().Info("validate delete", "name", in.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
