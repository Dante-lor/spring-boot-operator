package controller

import (
	"context"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	scalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("HPA Controller", func() {
	const resourceName = "test-hpa"
	const namespace = "default"

	var (
		ctx                  context.Context
		typeNamespacedName   types.NamespacedName
		controllerReconciler *SpringBootApplicationReconciler
		app                  *springv1alpha1.SpringBootApplication
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		app = &springv1alpha1.SpringBootApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: springv1alpha1.SpringBootApplicationSpec{
				Type:           springv1alpha1.SpringWeb,
				Image:          "test",
				ResourcePreset: ptr.To(v1alpha1.Small),
				Autoscaler: springv1alpha1.AutoscalingConfig{
					MinReplicas: 1,
					MaxReplicas: 5,
				},
			},
		}

		controllerReconciler = &SpringBootApplicationReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		By("creating the SpringBootApplication resource")
		Expect(k8sClient.Create(ctx, app)).To(Succeed())
	})

	AfterEach(func() {
		By("deleting the SpringBootApplication resource")
		Expect(k8sClient.Delete(ctx, app)).To(Succeed())
	})

	It("should create an HPA object with default values", func() {
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		hpa := &scalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(ctx, typeNamespacedName, hpa)).To(Succeed())

		Expect(*hpa.Spec.MinReplicas).To(Equal(int32(1)))
		Expect(hpa.Spec.MaxReplicas).To(Equal(int32(5)))
		Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(Equal(int32(70)))
	})

	It("should merge custom behavior with default behavior", func() {
		customBehavior := &scalingv2.HorizontalPodAutoscalerBehavior{
			ScaleUp: &scalingv2.HPAScalingRules{
				StabilizationWindowSeconds: ptr.To(int32(60)),
			},
		}
		app.Spec.Autoscaler.Behaviour = customBehavior
		Expect(k8sClient.Update(ctx, app)).To(Succeed())

		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		hpa := &scalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(ctx, typeNamespacedName, hpa)).To(Succeed())

		Expect(*hpa.Spec.Behavior.ScaleUp.StabilizationWindowSeconds).To(Equal(int32(60)))
		Expect(*hpa.Spec.Behavior.ScaleDown.StabilizationWindowSeconds).To(Equal(int32(300)))
	})
})
