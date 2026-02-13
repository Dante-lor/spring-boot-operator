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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
)

// SpringBootApplicationReconciler reconciles a SpringBootApplication object
type SpringBootApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SpringBootApplication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SpringBootApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	app := &v1alpha1.SpringBootApplication{}
	err := r.Get(ctx, req.NamespacedName, app)

	if (err != nil) {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling application", "name", app.ObjectMeta.Name, "namespace", app.ObjectMeta.Namespace)
	
	// 

	return ctrl.Result{}, nil
}

func (r *SpringBootApplicationReconciler) ensureConfigMap(ctx context.Context, app v1alpha1.SpringBootApplication) (error) {

	

	return nil;
}

func (r *SpringBootApplicationReconciler) ensureService(ctx context.Context, app v1alpha1.SpringBootApplication) (error) {

	

	return nil;
}

func (r *SpringBootApplicationReconciler) ensureDeployment(ctx context.Context, app v1alpha1.SpringBootApplication) (error) {



	return nil;
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpringBootApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&springv1alpha1.SpringBootApplication{}).
		Named("springbootapplication").
		Complete(r)
}
