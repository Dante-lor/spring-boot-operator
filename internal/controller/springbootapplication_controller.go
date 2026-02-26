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
	"encoding/json"
	"fmt"

	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	scalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// SpringBootApplicationReconciler reconciles a SpringBootApplication object
type SpringBootApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const EXTERNAL_PORT = 80

// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SpringBootApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	app := &springv1alpha1.SpringBootApplication{}
	err := r.Get(ctx, req.NamespacedName, app)

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling application", "name", app.Name, "namespace", app.Namespace)

	appConfig, err := mergeConfig(app.Spec)

	if err != nil {
		meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
			Type:               "Valid",
			Status:             metav1.ConditionFalse,
			Reason:             "FailedConfigMerge",
			Message:            err.Error(),
			ObservedGeneration: app.Generation,
		})
	} else {
		meta.SetStatusCondition(&app.Status.Conditions, metav1.Condition{
			Type:               "Valid",
			Status:             metav1.ConditionTrue,
			Reason:             "ConfigMergeSuccessful",
			Message:            "Generated Merged Spring Configuration",
			ObservedGeneration: app.Generation,
		})
	}

	// Try and update status
	if err := r.Status().Update(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureConfigMap(ctx, app, appConfig); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureService(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureDeployment(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.ensureAutoscaler(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Creates Configmap using provided string for the application.yaml file
func (r *SpringBootApplicationReconciler) ensureConfigMap(ctx context.Context, app *springv1alpha1.SpringBootApplication, config string) error {

	// Get existing configmap
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = app.Labels

		cm.Data = map[string]string{
			"application.yaml": config,
		}
		return controllerutil.SetControllerReference(app, cm, r.Scheme)
	})

	return err
}

// Creates HTTP service to handle web traffic
func (r *SpringBootApplicationReconciler) ensureService(ctx context.Context, app *springv1alpha1.SpringBootApplication) error {
	existing := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = app.Labels

		svc.Spec = corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       EXTERNAL_PORT,
					TargetPort: intstr.FromInt(app.Spec.Port),
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
		}

		return controllerutil.SetControllerReference(app, svc, r.Scheme)
	})

	return err
}

// mergeConfig merges the user provided configuration with the configuration defined on
// the spec (currently port and context path)
func mergeConfig(spec springv1alpha1.SpringBootApplicationSpec) (string, error) {
	// Step 1: unmarshal RawExtension JSON into a map
	merged := map[string]interface{}{}

	raw := spec.Config

	if raw != nil && len(raw.Raw) > 0 {
		if err := json.Unmarshal(raw.Raw, &merged); err != nil {
			return "", fmt.Errorf("failed to unmarshal RawExtension: %w", err)
		}
	}

	// Step 2: ensure "server" map exists
	server, ok := merged["server"].(map[string]interface{})
	if !ok {
		server = map[string]interface{}{}
	}

	// Step 3: set port and context path
	server["port"] = spec.Port

	servlet, ok := server["servlet"].(map[string]interface{})
	if !ok {
		servlet = map[string]interface{}{}
	}
	servlet["context-path"] = spec.ContextPath

	merged["server"] = server

	// Step 4: marshal merged map to YAML
	yamlBytes, err := yaml.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged config to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpringBootApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&springv1alpha1.SpringBootApplication{}).
		Named("springbootapplication").
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&scalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}
