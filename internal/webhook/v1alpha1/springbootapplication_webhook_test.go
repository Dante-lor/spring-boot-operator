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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
)

var _ = Describe("SpringBootApplication Webhook", func() {
	var (
		obj       *springv1alpha1.SpringBootApplication
		oldObj    *springv1alpha1.SpringBootApplication
		defaulter SpringBootApplicationResourceDefaulter
	)

	BeforeEach(func() {
		obj = &springv1alpha1.SpringBootApplication{}
		oldObj = &springv1alpha1.SpringBootApplication{}
		defaulter = SpringBootApplicationResourceDefaulter{}
		Expect(defaulter).NotTo(BeNil(), "Expected defaulter to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	Context("When creating SpringBootApplication under Defaulting Webhook", func() {

		It("Should reject other objects", func() {
			other := &appsv1.Deployment{}
			Expect(defaulter.Default(ctx, other)).NotTo(Succeed())
		})

		presets := []v1alpha1.ResourcePreset{
			v1alpha1.Small,
			v1alpha1.Medium,
			v1alpha1.Large,
		}

		for _, element := range presets {

			It(fmt.Sprintf("should leave preset if already defined as %s", element), func() {

				obj.Spec.ResourcePreset = &element

				defaulter.Default(ctx, obj)

				Expect(*obj.Spec.ResourcePreset).To(Equal(element))
			})

			It(fmt.Sprintf("should remove preset of %s if Resources are defined", element), func() {
				obj.Spec.ResourcePreset = &element
				obj.Spec.Resources = &v1alpha1.ResourceDefinition{}

				defaulter.Default(ctx, obj)

				Expect(obj.Spec.ResourcePreset).To(BeNil())
			})

		}

		It("should leave preset as nil if resources are defined", func() {
			obj.Spec.Resources = &v1alpha1.ResourceDefinition{}

			defaulter.Default(ctx, obj)

			Expect(obj.Spec.ResourcePreset).To(BeNil())
		})

		It("should set preset to small if neither resources, nor preset are defined", func() {
			defaulter.Default(ctx, obj)

			Expect(*obj.Spec.ResourcePreset).To(Equal(v1alpha1.Small))
		})
	})

})
