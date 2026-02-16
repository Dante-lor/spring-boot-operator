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
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
)

// SpringBootApplicationReconciler reconciles a SpringBootApplication object
type SpringBootApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Used internally
const DEFAULT_INTERNAL_PORT = 8080
const EXTERNAL_PORT = 80

// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=spring.dante-lor.github.io,resources=springbootapplications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SpringBootApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	app := &v1alpha1.SpringBootApplication{}
	err := r.Get(ctx, req.NamespacedName, app)

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling application", "name", app.ObjectMeta.Name, "namespace", app.ObjectMeta.Namespace)

	appConfig, internalPort, err := mergeConfigWithDefaultPort(app.Spec.Config)

	r.ensureConfigMap(ctx, app, appConfig);
	r.ensureService(ctx, app, internalPort);
	r.ensureDeployment(ctx, app, internalPort);

	return ctrl.Result{}, nil
}

// Creates Configmap using provided string for the application.yaml file
func (r *SpringBootApplicationReconciler) ensureConfigMap(ctx context.Context, app *v1alpha1.SpringBootApplication, config string) error {

	// Get existing configmap
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Data: map[string]string{
			"application.yaml": config,
		},
	}

	// If error exists (not found)
	if err != nil {
		err = r.Create(ctx, desired)

	// If they are not equal update to desired state
	} else if !reflect.DeepEqual(desired.Data, existing.Data) {
		err = r.Update(ctx, desired)
	}

	return err;
}

// Creates HTTP service to handle web traffic
func (r *SpringBootApplicationReconciler) ensureService(ctx context.Context, app *v1alpha1.SpringBootApplication, internalPort int) error {
	logger := ctrl.LoggerFrom(ctx)
	existing := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       EXTERNAL_PORT,
					TargetPort: intstr.FromInt(internalPort),
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}

	// If there was an error, it means no config map exists
	if err != nil {
		// Service doesn't exist → create

		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		logger.Info("Created Service", "name", app.Name)
	} else {
		// Service exists → check if it needs update
		// Only update fields that are mutable (Type, Ports, Selector are mutable in most clusters)
		needsUpdate := false

		// Compare ports
		if !reflect.DeepEqual(existing.Spec.Ports, desired.Spec.Ports) {
			existing.Spec.Ports = desired.Spec.Ports
			needsUpdate = true
		}

		// Compare selector
		if !reflect.DeepEqual(existing.Spec.Selector, desired.Spec.Selector) {
			existing.Spec.Selector = desired.Spec.Selector
			needsUpdate = true
		}

		// Compare type
		if (existing.Spec.Type != desired.Spec.Type) {
			existing.Spec.Type = desired.Spec.Type;
			needsUpdate = true
		}

		if needsUpdate {
			if err := r.Update(ctx, existing); err != nil {
				return err
			}
			logger.Info("Updated Service", "name", app.Name)
		}
	}

	return nil
}

func (r *SpringBootApplicationReconciler) ensureDeployment(ctx context.Context, app *v1alpha1.SpringBootApplication, internalPort int) error {
	existing := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if (client.IgnoreNotFound(err) != nil) {
		return err;
	} else if err != nil {
		// Not found - create it
		desired, err := r.createDeploymentObject(app, internalPort);
		if err != nil {
			return err;
		}
		return r.Create(ctx, &desired)
	} else {
		// Found so update if it has changed
		desired, err := r.createDeploymentObject(app, internalPort);
		if err != nil {
			return err;
		}

		controllerutil.SetControllerReference(app, &desired, r.Scheme);
		// Need to compare existing deployment specs
		if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) {
			r.Update(ctx, &desired)
		}

	}



	return nil
}

func (r *SpringBootApplicationReconciler) createDeploymentObject(app *v1alpha1.SpringBootApplication, internalPort int) (appsv1.Deployment, error) {
	labels := app.GetLabels()
	labels["app"] = app.Name

	// Try and create resources from the app object

	resources, err := createResources(*app);

	if (err != nil) {
		return appsv1.Deployment{}, err;
	}

	replicas := int32(1);

	runAsNonRoot := true;
	allowPriviledgeEscalation := false
	readOnlyFileSystem := true

	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							Image: app.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name: "http",
									ContainerPort: int32(internalPort),
								},
							},
							Resources: resources,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPriviledgeEscalation,
								ReadOnlyRootFilesystem: &readOnlyFileSystem,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",	
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									Items: []corev1.KeyToPath{
										{
											Key: app.Name,
										},
									},
								},
							},
						},
					},
				},

			},
		},
	}

	if err := controllerutil.SetControllerReference(app, &dep, r.Scheme); err != nil {
    	return dep, err
	}

	return dep, nil;
}


func createResources (app springv1alpha1.SpringBootApplication) (corev1.ResourceRequirements, error) {
	switch *app.Spec.ResourcePreset {
		case v1alpha1.Small:
			return createSpringResourceRequirements(resource.MustParse("1"), resource.MustParse("1Gi")), nil
		case v1alpha1.Medium:
			return createSpringResourceRequirements(resource.MustParse("2"), resource.MustParse("2Gi")), nil
		case v1alpha1.Large:
			return createSpringResourceRequirements(resource.MustParse("4"), resource.MustParse("4Gi")), nil
		default:
			// Try and parse
			cpu, err := resource.ParseQuantity(app.Spec.Resources.CPU)
			if (err != nil) {
				return corev1.ResourceRequirements{}, err;
			}

			memory, err := resource.ParseQuantity(app.Spec.Resources.Memory)

			if (err != nil) {
				return corev1.ResourceRequirements{}, err
			}

			return createSpringResourceRequirements(cpu, memory), nil;
	}
}

func createSpringResourceRequirements(cpu resource.Quantity, memory resource.Quantity) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: cpu,
			corev1.ResourceMemory: memory,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: memory,
		},
	}
}

// mergeConfigWithDefaultPort merges the RawExtension config with a default server port
// If the user already specifies server.port, it keeps that value.
// Returns the merged YAML string and the port as int.
func mergeConfigWithDefaultPort(raw *runtime.RawExtension) (string, int, error) {
	// Step 1: unmarshal RawExtension JSON into a map
	merged := map[string]interface{}{}

	if raw != nil && len(raw.Raw) > 0 {
		if err := json.Unmarshal(raw.Raw, &merged); err != nil {
			return "", 0, fmt.Errorf("failed to unmarshal RawExtension: %w", err)
		}
	}

	// Step 2: ensure "server" map exists
	server, ok := merged["server"].(map[string]interface{})
	if !ok {
		server = map[string]interface{}{}
	}

	// Step 3: check if port is set, if not, set default
	port, ok := server["port"].(int) // JSON numbers come back as float64
	if !ok {
		if portFloat, ok := server["port"].(float64); ok {
			port = int(portFloat)
		} else {
			port = DEFAULT_INTERNAL_PORT
		}
	}

	// If port was not defined, inject default
	if _, exists := server["port"]; !exists {
		server["port"] = port
	}

	merged["server"] = server

	// Step 4: marshal merged map to YAML
	yamlBytes, err := yaml.Marshal(merged)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal merged config to YAML: %w", err)
	}

	return string(yamlBytes), port, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpringBootApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&springv1alpha1.SpringBootApplication{}).
		Named("springbootapplication").
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
