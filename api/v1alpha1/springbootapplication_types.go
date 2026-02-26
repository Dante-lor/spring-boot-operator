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
	scalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourcePreset string

const (
	Small  ResourcePreset = "small"
	Medium ResourcePreset = "medium"
	Large  ResourcePreset = "large"
)

type SpringFramework string

const (
	SpringWeb     SpringFramework = "web"
	SpringWebflux SpringFramework = "webflux"
	SpringNative                  = "native"
)

// Describes cpu and memory requirements for a spring application to run when under normal load
// If ResourcePreset is not used, a user must specify CPU and memory usage
type ResourceDefinition struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// Utilization Percentage
type UtilizationTarget struct {

	// CPU percentage target
	CpuPercentage *int32 `json:"cpuPercentage,omitempty"`
}

// Custom Autoscaling configuration.
type AutoscalingConfig struct {

	// +kubebuilder:default=2
	// Min replicas
	MinReplicas int `json:"minReplicas,omitempty"`

	// +kubebuilder:default=10
	// Max replicas
	MaxReplicas int `json:"maxReplicas,omitempty"`

	// Utilization target
	TargetUtilization UtilizationTarget `json:"utilizationTarget,omitempty"`

	// Scaling behaviour
	Behaviour *scalingv2.HorizontalPodAutoscalerBehavior `json:"behaviour,omitempty"`
}

// SpringBootApplicationSpec defines the desired state of SpringBootApplication.
type SpringBootApplicationSpec struct {
	// +kubebuilder:validation:MinLength=1
	// Docker image to run (required)
	Image string `json:"image"`

	// +kubebuilder:validation:Enum=web;webflux;native
	// +kubebuilder:default=web
	// Type of Spring Boot Application
	Type SpringFramework `json:"type,omitempty"`

	// +kubebuilder:default=8080
	// Internal HTTP port to use
	Port int `json:"port,omitempty"`

	// +kubebuilder:default=/
	// Context path for the application to use
	ContextPath string `json:"contextPath,omitempty"`

	// Application.yaml file contents
	Config *runtime.RawExtension `json:"config,omitempty"`

	// +kubebuilder:validation:Enum=small;medium;large
	// Resource preset
	ResourcePreset *ResourcePreset `json:"resourcePreset,omitempty"`

	// Custom resources object - you can use this instead of using the preset.
	Resources *ResourceDefinition `json:"resources,omitempty"`

	// Autoscaling configuration
	Autoscaler AutoscalingConfig `json:"autoscaler,omitempty"`
}

// SpringBootApplicationStatus defines the observed state of SpringBootApplication.
type SpringBootApplicationStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpringBootApplication is the Schema for the springbootapplications API.
type SpringBootApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpringBootApplicationSpec   `json:"spec"`
	Status SpringBootApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpringBootApplicationList contains a list of SpringBootApplication.
type SpringBootApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpringBootApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpringBootApplication{}, &SpringBootApplicationList{})
}
