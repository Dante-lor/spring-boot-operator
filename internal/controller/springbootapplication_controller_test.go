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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	res "k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
)

var _ = Describe("SpringBootApplication Controller", func() {
	const resourceName = "test-resource"

	ctx := context.Background()

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}

	var controllerReconciler *SpringBootApplicationReconciler

	Context("When reconciling an empty resource", func() {
		BeforeEach(func() {
			By("creating the custom resource for the Kind SpringBootApplication")
			resource := &springv1alpha1.SpringBootApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: springv1alpha1.SpringBootApplicationSpec{
					Image: "test",
				},
			}
			controllerReconciler = &SpringBootApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			springbootapplication := &springv1alpha1.SpringBootApplication{}
			err := k8sClient.Get(ctx, typeNamespacedName, springbootapplication)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

		})

		AfterEach(func() {
			resource := &springv1alpha1.SpringBootApplication{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SpringBootApplication")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should fail to reconcile the resource", func() {
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})

		It("should not retry", func() {
			res, _ := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(res).ToNot(BeNil())
			Expect(res.RequeueAfter).To(BeEquivalentTo(0))
		})
	})

	Context("When reconciling a minimal valid application", func() {

		var resource *springv1alpha1.SpringBootApplication

		CheckExpectedPresetBehaviour := func(preset *springv1alpha1.ResourcePreset, expectedCPUStr string, expectedMemoryStr string) {
			resource.Spec.ResourcePreset = preset

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			dep := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedName, dep)
			Expect(err).NotTo(HaveOccurred())

			expectedCPU := res.MustParse(expectedCPUStr);
			expectedMem := res.MustParse(expectedMemoryStr);

			deployResources := dep.Spec.Template.Spec.Containers[0].Resources

			Expect(*deployResources.Requests.Cpu()).To(BeIdenticalTo(expectedCPU))
			Expect(*deployResources.Requests.Memory()).To(BeIdenticalTo(expectedMem))
			Expect(*deployResources.Limits.Memory()).To(BeIdenticalTo(expectedMem))

			// Checking the CPU would yeild an empty resource value. Therefore checking the length to
			// Ensure only one value is set (the memory as previously tested)
			Expect(len(deployResources.Limits)).To(BeIdenticalTo(1))
		}

		BeforeEach(func() {
			resource = &springv1alpha1.SpringBootApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: springv1alpha1.SpringBootApplicationSpec{
					Image: "test",
				},
			}
			controllerReconciler = &SpringBootApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		AfterEach(func() {
			resource := &springv1alpha1.SpringBootApplication{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance SpringBootApplication")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("sets resource requests and memory limits for small preset", func() {
			resourcePreset := v1alpha1.Small
			CheckExpectedPresetBehaviour(&resourcePreset, "1", "1Gi")
		})

		It("sets resource requests and memory limits for medium preset", func() {
			resourcePreset := v1alpha1.Medium
			CheckExpectedPresetBehaviour(&resourcePreset, "2", "2Gi")
		})

		It("sets resource requests and memory limits for large preset", func() {
			resourcePreset := v1alpha1.Large
			CheckExpectedPresetBehaviour(&resourcePreset, "4", "4Gi")
		})

		It("uses custom resources if defined and preset is nil", func ()  {
			resource.Spec.Resources = &springv1alpha1.ResourceDefinition{
				CPU: "2",
				Memory: "8Gi",
			}

			CheckExpectedPresetBehaviour(nil, "2", "8Gi");
		})
	})

	
})
