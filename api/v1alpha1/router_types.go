/*
Copyright 2024.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RouterSpec defines the desired state of Router
type RouterSpec struct {

	// List of VLLMRuntime references
	Runtimes []RuntimeReference `json:"runtimes"`

	// Load balancing strategy
	// +kubebuilder:validation:Enum=roundrobin;session
	// +kubebuilder:default=roundrobin
	RoutingLogic string `json:"routingLogic,omitempty"`

	// Service discovery
	// +kubebuilder:validation:Enum=k8s;static
	// +kubebuilder:default=k8s
	ServiceDiscovery string `json:"serviceDiscovery,omitempty"`

	// Resource requirements
	Resources ResourceRequirements `json:"resources"`

	// Image configuration
	Image ImageSpec `json:"image"`

	// Replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Environment variables
	Env []EnvVar `json:"env,omitempty"`

	// Extra arguments
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

// RuntimeReference defines a reference to a VLLMRuntime
type RuntimeReference struct {
	// Name of the VLLMRuntime
	Name string `json:"name"`

	// Namespace of the VLLMRuntime
	Namespace string `json:"namespace"`

	// Weight for load balancing
	Weight int32 `json:"weight,omitempty"`
}

// RouterStatus defines the observed state of Router
type RouterStatus struct {
	// Router status
	Status string `json:"status,omitempty"`

	// Last updated timestamp
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// Number of active runtimes
	ActiveRuntimes int32 `json:"activeRuntimes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Router is the Schema for the routers API
type Router struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RouterSpec   `json:"spec,omitempty"`
	Status RouterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RouterList contains a list of Router
type RouterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Router `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Router{}, &RouterList{})
}
