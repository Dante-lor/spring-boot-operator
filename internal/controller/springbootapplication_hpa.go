package controller

import (
	"context"
	"fmt"

	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	scalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *SpringBootApplicationReconciler) ensureAutoscaler(ctx context.Context, app *springv1alpha1.SpringBootApplication) error {

	// Get existing configmap
	existing := &scalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, client.ObjectKeyFromObject(app), existing)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	hpa := &scalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
		// Set labels
		hpa.Labels = app.Labels

		hpa.Spec, err = createAutoscalerSpec(app)

		if err != nil {
			return err
		}

		return controllerutil.SetControllerReference(app, hpa, r.Scheme)
	})

	return err
}

func createAutoscalerSpec(app *springv1alpha1.SpringBootApplication) (scalingv2.HorizontalPodAutoscalerSpec, error) {
	spec := scalingv2.HorizontalPodAutoscalerSpec{}

	config := app.Spec.Autoscaler

	// Target Ref
	spec.ScaleTargetRef = scalingv2.CrossVersionObjectReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       app.Name,
	}

	// Replicas
	spec.MinReplicas = ptr.To(int32(config.MinReplicas))
	spec.MaxReplicas = int32(config.MaxReplicas)

	// CPU utilization
	averageTargetPercent := config.TargetUtilization.CpuPercentage

	if averageTargetPercent == nil {
		switch app.Spec.Type {
		case springv1alpha1.SpringWeb:
			averageTargetPercent = ptr.To(int32(70))
		case springv1alpha1.SpringWebflux:
			averageTargetPercent = ptr.To(int32(75))
		case springv1alpha1.SpringNative:
			averageTargetPercent = ptr.To(int32(65))
		default:
			return spec, fmt.Errorf("unrecognized application type: %s", app.Spec.Type)
		}
	}

	spec.Metrics = []scalingv2.MetricSpec{
		{
			Type: scalingv2.ResourceMetricSourceType,
			Resource: &scalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: scalingv2.MetricTarget{
					Type:               scalingv2.UtilizationMetricType,
					AverageUtilization: averageTargetPercent,
				},
			},
		},
	}

	// Scaling Behavior
	defaultBehaviour, err := getDefaultBehaviour(app.Spec.Type)

	if err != nil {
		return spec, err
	}

	// Custom user defined scaling behaviour
	custom := config.Behaviour

	if custom == nil {
		spec.Behavior = defaultBehaviour
	} else {
		spec.Behavior = mergeBehaviours(custom, defaultBehaviour)
	}
	return spec, nil
}

func mergeBehaviours(custom *scalingv2.HorizontalPodAutoscalerBehavior, defaultBehaviour *scalingv2.HorizontalPodAutoscalerBehavior) *scalingv2.HorizontalPodAutoscalerBehavior {

	if custom.ScaleUp == nil {
		custom.ScaleUp = defaultBehaviour.ScaleUp
	}

	if custom.ScaleDown == nil {
		custom.ScaleDown = defaultBehaviour.ScaleDown
	}

	return custom
}

func createBehaviour(scaleUpStabilization int, scaleUpPercentage int, scaleUpPeriodSeconds int, scaleDownStabilization int) *scalingv2.HorizontalPodAutoscalerBehavior {

	return &scalingv2.HorizontalPodAutoscalerBehavior{
		ScaleUp: &scalingv2.HPAScalingRules{
			StabilizationWindowSeconds: ptr.To(int32(scaleUpStabilization)),
			Policies: []scalingv2.HPAScalingPolicy{
				{
					Type:          scalingv2.PercentScalingPolicy,
					Value:         int32(scaleUpPercentage),
					PeriodSeconds: int32(scaleUpPeriodSeconds),
				},
			},
		},
		ScaleDown: &scalingv2.HPAScalingRules{
			StabilizationWindowSeconds: ptr.To(int32(scaleDownStabilization)),
		},
	}

}

func getDefaultBehaviour(springFramework springv1alpha1.SpringFramework) (*scalingv2.HorizontalPodAutoscalerBehavior, error) {
	switch springFramework {
	case springv1alpha1.SpringWeb:
		return createBehaviour(30, 50, 60, 300), nil
	case springv1alpha1.SpringWebflux:
		return createBehaviour(20, 75, 60, 240), nil
	case springv1alpha1.SpringNative:
		return createBehaviour(10, 100, 30, 120), nil
	}

	return nil, fmt.Errorf("unhandled spring framework: %s", springFramework)
}
