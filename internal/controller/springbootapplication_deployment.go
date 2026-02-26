package controller

import (
	"context"
	"fmt"

	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *SpringBootApplicationReconciler) ensureDeployment(ctx context.Context, app *springv1alpha1.SpringBootApplication) error {
	existing := &appsv1.Deployment{}

	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		desired, err := r.createDeploymentObject(app)

		if err != nil {
			return err
		}

		deploy.Labels = desired.Labels
		deploy.Spec = desired.Spec

		return controllerutil.SetControllerReference(app, deploy, r.Scheme)
	})

	return err
}

func (r *SpringBootApplicationReconciler) createDeploymentObject(app *springv1alpha1.SpringBootApplication) (appsv1.Deployment, error) {
	labels := app.GetLabels()

	if labels == nil {
		labels = map[string]string{
			"app": app.Name,
		}
	} else {
		labels["app"] = app.Name
	}

	// Try and create resources from the app object

	resources, err := createResources(*app)

	if err != nil {
		return appsv1.Deployment{}, err
	}

	runAsNonRoot := true
	allowPriviledgeEscalation := false
	readOnlyFileSystem := true

	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil, // This is nil because the Autoscaler will set the replicas
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
							Name:  "app",
							Image: app.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: int32(app.Spec.Port),
								},
							},
							Resources: resources,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPriviledgeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyFileSystem,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									// Using additional config means that this config is merged with their existing
									// Configuration, meaning the config object doesn't have to be as large
									Name:  "SPRING_CONFIG_ADDITIONAL_LOCATION",
									Value: "/config",
								},
								{
									// By default, java only uses 25% of it's memory for the java heap. That is quite low
									// Setting this to 70 allows it to use more. Some needs to be left for GC.
									Name:  "JAVA_TOOL_OPTIONS",
									Value: "-XX:MaxRAMPercentage=70",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: app.Name,
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

	return dep, nil
}

func createResources(app springv1alpha1.SpringBootApplication) (corev1.ResourceRequirements, error) {
	if app.Spec.ResourcePreset == nil {
		resources := app.Spec.Resources
		if resources == nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("either resource preset or resource must be defined")
		}

		// Try and parse
		cpu, err := resource.ParseQuantity(resources.CPU)
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}

		memory, err := resource.ParseQuantity(resources.Memory)

		if err != nil {
			return corev1.ResourceRequirements{}, err
		}

		return createSpringResourceRequirements(cpu, memory), nil
	}

	switch *app.Spec.ResourcePreset {
	case springv1alpha1.Small:
		return createSpringResourceRequirements(resource.MustParse("1"), resource.MustParse("1Gi")), nil
	case springv1alpha1.Medium:
		return createSpringResourceRequirements(resource.MustParse("2"), resource.MustParse("2Gi")), nil
	case springv1alpha1.Large:
		return createSpringResourceRequirements(resource.MustParse("4"), resource.MustParse("4Gi")), nil
	default:
		return corev1.ResourceRequirements{}, fmt.Errorf("unrecognized resource preset: %s", *app.Spec.ResourcePreset)
	}
}

func createSpringResourceRequirements(cpu resource.Quantity, memory resource.Quantity) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: memory,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: memory,
		},
	}
}
