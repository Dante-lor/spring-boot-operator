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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	Context("When reconciling a valid application", func() {

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

			expectedCPU := res.MustParse(expectedCPUStr)
			expectedMem := res.MustParse(expectedMemoryStr)

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

		It("uses custom resources if defined and preset is nil", func() {
			resource.Spec.Resources = &springv1alpha1.ResourceDefinition{
				CPU:    "2",
				Memory: "8Gi",
			}

			CheckExpectedPresetBehaviour(nil, "2", "8Gi")
		})

		Describe("with small preset", func () {

			BeforeEach(func() {
				resourcePreset := v1alpha1.Small
				resource.Spec.ResourcePreset = &resourcePreset
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})

				Expect(err).NotTo(HaveOccurred())
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

			It("mounts the config at /config", func() {
				deploy := &appsv1.Deployment{}
	
				Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

				volumeName := "config";
				// Check volume is added
				volumes := deploy.Spec.Template.Spec.Volumes
				Expect(volumes).To(HaveLen(1))
				vol := volumes[0]
				Expect(vol.Name).To(Equal(volumeName))
				Expect(vol.ConfigMap.LocalObjectReference.Name).To(Equal(resource.Name))

				// Check it's mounted at /config
				mounts := deploy.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(mounts).To(HaveLen(1))
				configMount := mounts[0]
				Expect(configMount.Name).To(Equal(volumeName))
				Expect(configMount.MountPath).To(Equal("/config"))
			})

			It("adds environment variable to tell where additional properties are located", func() {
				deploy := &appsv1.Deployment{}
	
				Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())
				env := deploy.Spec.Template.Spec.Containers[0].Env

				Expect(env).To(HaveLen(2))
				configEnvVar := env[0]

				Expect(configEnvVar.Name).To(Equal("SPRING_CONFIG_ADDITIONAL_LOCATION"))
				Expect(configEnvVar.Value).To(Equal("/config"))
			})

			It("uses 70 percent of available memory for the java heap", func() {
				deploy := &appsv1.Deployment{}
	
				Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())
				env := deploy.Spec.Template.Spec.Containers[0].Env

				Expect(env).To(HaveLen(2))
				configEnvVar := env[1]

				Expect(configEnvVar.Name).To(Equal("JAVA_TOOL_OPTIONS"))
				Expect(configEnvVar.Value).To(Equal("-XX:MaxRAMPercentage=70"))
			})
		})

		Describe("when the port is overriden", func() {
			BeforeEach(func() {
				resource.Spec.ResourcePreset = ptr.To(v1alpha1.Small)

				config := map[string]any{
					"server": map[string]any{
						"port": 3333, // Different port
					},
				}

				raw, _ := json.Marshal(config)

				resource.Spec.Config = &runtime.RawExtension{
					Raw: raw,
				}

				Expect(k8sClient.Create(ctx, resource)).To(Succeed())

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

			It("Uses the updated port in the service", func() {
				svc := &corev1.Service{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, svc)).To(Succeed())

				Expect(svc.Spec.Ports).To(HaveLen(1))
				port := svc.Spec.Ports[0]

				Expect(port.TargetPort).To(Equal(intstr.FromInt(3333)))
			})

			It("Exposes the port in the deployment", func ()  {
				deploy := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

				ports := deploy.Spec.Template.Spec.Containers[0].Ports

				Expect(ports).To(HaveLen(1))
				exposed := ports[0]

				Expect(exposed.ContainerPort).To(BeEquivalentTo(3333))
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

			BeforeEach(func() {
				resourcePreset := v1alpha1.Small
				resource.Spec.ResourcePreset = &resourcePreset

				Expect(k8sClient.Create(ctx, resource)).To(Succeed())

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				// Refresh the in memory UID
				Expect(k8sClient.Get(ctx, typeNamespacedName, resource)).To(Succeed())
			})

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
