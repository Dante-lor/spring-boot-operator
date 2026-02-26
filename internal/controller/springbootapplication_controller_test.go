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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	Context("When reconciling a valid application", func() {

		var resource *springv1alpha1.SpringBootApplication

		BeforeEach(func() {

			resource = &springv1alpha1.SpringBootApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: springv1alpha1.SpringBootApplicationSpec{
					Image:          "test",
					ResourcePreset: ptr.To(v1alpha1.Small),
				},
			}

			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			// Refresh the in memory UID
			Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("sets application.yaml to minimal config including port", func() {
			cm := &corev1.ConfigMap{}

			Expect(k8sClient.Get(ctx, typeNamespacedName, cm)).To(Succeed())

			Expect(cm.Data).To(HaveLen(1))
			Expect(cm.Data).To(HaveKey("application.yaml"))

			configFileData := cm.Data["application.yaml"]

			expected :=
				`server:
  port: 8080
`
			Expect(configFileData).To(Equal(expected))
		})

		Describe("when the port is overridden", func() {
			BeforeEach(func() {

				resource.Spec.Port = 3333

				Expect(k8sClient.Update(ctx, resource)).To(Succeed())

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})

				Expect(err).NotTo(HaveOccurred())
			})

			It("sets the config correctly", func() {
				cm := &corev1.ConfigMap{}

				Expect(k8sClient.Get(ctx, typeNamespacedName, cm)).To(Succeed())

				Expect(cm.Data).To(HaveLen(1))
				Expect(cm.Data).To(HaveKey("application.yaml"))

				configFileData := cm.Data["application.yaml"]

				expected :=
					`server:
  port: 3333
`
				Expect(configFileData).To(Equal(expected))
			})

			It("uses the updated port in the service", func() {
				svc := &corev1.Service{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, svc)).To(Succeed())

				Expect(svc.Spec.Ports).To(HaveLen(1))
				port := svc.Spec.Ports[0]

				Expect(port.TargetPort).To(Equal(intstr.FromInt(3333)))
			})
		})

		Describe("OwnerReferences on Sub-Resources", func() {

			SubResourceHasOwnerReference := func(sub client.Object) {

				err := k8sClient.Get(ctx, typeNamespacedName, sub)
				Expect(err).NotTo(HaveOccurred())

				Expect(sub.GetOwnerReferences()).To(HaveLen(1))
				Expect(sub.GetOwnerReferences()).To(HaveExactElements(metav1.OwnerReference{
					APIVersion:         "spring.dante-lor.github.io/v1alpha1",
					Kind:               "SpringBootApplication",
					UID:                resource.UID,
					Name:               resource.Name,
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				}))
			}

			It("should set ownerReference on deployment", func() {
				SubResourceHasOwnerReference(&appsv1.Deployment{})
			})

			It("should set ownerReference on configmap", func() {
				SubResourceHasOwnerReference(&corev1.ConfigMap{})
			})

			It("should set ownerReference on service", func() {
				SubResourceHasOwnerReference(&corev1.Service{})
			})
		})
	})
})
