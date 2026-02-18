/*
Copyright 2026 Daniel Taylor.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
)

// log is for logging in this package.
var springbootapplicationlog = logf.Log.WithName("springbootapplication-resource")

// SetupSpringBootApplicationWebhookWithManager registers the webhook for SpringBootApplication in the manager.
func SetupSpringBootApplicationWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&springv1alpha1.SpringBootApplication{}).
		WithDefaulter(&SpringBootApplicationResourceDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-spring-dante-lor-github-io-v1alpha1-springbootapplication,mutating=true,failurePolicy=fail,sideEffects=None,groups=spring.dante-lor.github.io,resources=springbootapplications,verbs=create;update,versions=v1alpha1,name=mspringbootapplication-v1alpha1.kb.io,admissionReviewVersions=v1

// SpringBootApplicationResourceDefaulter struct is responsible for setting default values on the custom resource of the
// Kind SpringBootApplication when those are created or updated.
type SpringBootApplicationResourceDefaulter struct {
}

var _ webhook.CustomDefaulter = &SpringBootApplicationResourceDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind SpringBootApplication.
func (d *SpringBootApplicationResourceDefaulter) Default(_ context.Context, obj runtime.Object) error {
	springbootapplication, ok := obj.(*springv1alpha1.SpringBootApplication)

	if !ok {
		return fmt.Errorf("expected an SpringBootApplication object but got %T", obj)
	}
	springbootapplicationlog.Info("Defaulting for SpringBootApplication", "name", springbootapplication.GetName())

	if springbootapplication.Spec.Resources != nil {
		springbootapplication.Spec.ResourcePreset = nil
	} else if springbootapplication.Spec.ResourcePreset == nil {
		springbootapplication.Spec.ResourcePreset = ptr.To(v1alpha1.Small)
	}

	return nil
}
