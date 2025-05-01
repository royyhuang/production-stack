/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProductionStackSpec defines the desired state of ProductionStack.
type ProductionStackSpec struct {
	// Model defines the model configuration
	Model ModelSpec `json:"model"`

	// VLLMConfig defines the vLLM-specific configuration
	VLLMConfig VLLMConfig `json:"vllmConfig,omitempty"`

	// LMCacheConfig defines the LM Cache configuration
	LMCacheConfig LMCacheConfig `json:"lmcacheConfig,omitempty"`

	// RouterConfig defines the router configuration
	RouterConfig RouterConfig `json:"routerConfig,omitempty"`

	// Replicas is the number of model serving replicas
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// Resources defines the resource requirements for each replica
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Strategy defines the deployment strategy to use
	// +kubebuilder:validation:Enum=RollingUpdate;Recreate
	// +kubebuilder:default=RollingUpdate
	DeploymentStrategy string `json:"deploymentStrategy,omitempty"`

	// Env is a list of environment variables to set
	Env []EnvVar `json:"env,omitempty"`
}

// ModelSpec defines the configuration for the vLLM model
type ModelSpec struct {
	// Name is the name of the model to serve
	Name string `json:"name"`

	// Path is the path to the model files
	Path string `json:"path,omitempty"`

	// ModelURL is the URL to download the model from
	ModelURL string `json:"modelURL,omitempty"`

	// EnableLoRA enables LoRA support
	EnableLoRA bool `json:"enableLoRA,omitempty"`

	// EnableTool enables automatic tool choice
	EnableTool bool `json:"enableTool,omitempty"`

	// ToolCallParser specifies the tool call parser to use
	ToolCallParser string `json:"toolCallParser,omitempty"`

	// ChatTemplate specifies the chat template to use
	ChatTemplate string `json:"chatTemplate,omitempty"`

	// TrustRemoteCode enables loading custom model code
	TrustRemoteCode bool `json:"trustRemoteCode,omitempty"`

	// DType is the data type to use
	// +kubebuilder:validation:Enum=float16;bfloat16;float32
	DType string `json:"dtype,omitempty"`

	// MaxNumSeqs is the maximum number of sequences to process in parallel
	MaxNumSeqs int32 `json:"maxNumSeqs,omitempty"`

	// MaxModelLen is the maximum model length
	MaxModelLen int32 `json:"maxModelLen,omitempty"`
}

// ResourceRequirements defines the resource requirements for a vLLM replica
type ResourceRequirements struct {
	// CPU is the CPU resource requirement
	CPU string `json:"cpu,omitempty"`

	// Memory is the memory resource requirement
	Memory string `json:"memory,omitempty"`

	// GPU is the GPU resource requirement
	// +kubebuilder:default="1"
	GPU string `json:"gpu,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	// Name is the name of the environment variable
	Name string `json:"name"`

	// Value is the value of the environment variable
	// +optional
	Value string `json:"value,omitempty"`
}

// VLLMConfig defines the vLLM-specific configuration
type VLLMConfig struct {
	// Image defines the image configuration
	Image Image `json:"image,omitempty"`

	// EnableChunkedPrefill enables chunked prefill
	EnableChunkedPrefill bool `json:"enableChunkedPrefill,omitempty"`

	// EnablePrefixCaching enables prefix caching
	EnablePrefixCaching bool `json:"enablePrefixCaching,omitempty"`

	// TensorParallelSize is the number of GPUs to use for tensor parallelism
	TensorParallelSize int32 `json:"tensorParallelSize,omitempty"`

	// GpuMemoryUtilization is the target GPU memory utilization (0.0 to 1.0)
	// +kubebuilder:validation:Pattern=^0(\.[0-9]+)?|1(\.0+)?$
	GpuMemoryUtilization string `json:"gpuMemoryUtilization,omitempty"`

	// MaxLoras is the maximum number of LoRAs to support
	MaxLoras int32 `json:"maxLoras,omitempty"`

	// HFTokenSecretName is the name of the secret to use for pulling the image from Hugging Face
	HFTokenSecret corev1.LocalObjectReference `json:"hfTokenSecret,omitempty"`

	// ExtraArgs are additional command-line arguments to pass to vLLM
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// V1 enables v1 compatibility mode
	V1 bool `json:"v1,omitempty"`
}

// Image defines the image configuration
type Image struct {
	// Name is the Docker image to use
	// +kubebuilder:default="vllm/vllm-openai:latest"
	Name string `json:"name,omitempty"`

	// Registry is the registry to use for pulling the image
	// +kubebuilder:default="docker.io"
	Registry string `json:"registry,omitempty"`

	// PullPolicy is the policy for pulling the image
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default=IfNotPresent
	PullPolicy string `json:"pullPolicy,omitempty"`

	// PullSecretName is the name of the secret to use for pulling the image
	PullSecretName string `json:"pullSecretName,omitempty"`
}

// LMCacheConfig defines the LM Cache configuration
type LMCacheConfig struct {
	// Enabled enables LM Cache
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// CPUOffloadingBufferSize is the size of the CPU offloading buffer
	// +kubebuilder:default="4Gi"
	CPUOffloadingBufferSize string `json:"cpuOffloadingBufferSize,omitempty"`

	// DiskOffloadingBufferSize is the size of the disk offloading buffer
	// +kubebuilder:default="8Gi"
	DiskOffloadingBufferSize string `json:"diskOffloadingBufferSize,omitempty"`

	// RemoteURL is the URL of the remote cache server
	RemoteURL string `json:"remoteUrl,omitempty"`

	// RemoteSerde is the serialization format for the remote cache
	RemoteSerde string `json:"remoteSerde,omitempty"`
}

type RouterConfig struct {
	// RouterType is the type of router to use
	// +kubebuilder:validation:Enum=roundrobin;session
	// +kubebuilder:default=roundrobin
	RouterType string `json:"routerType,omitempty"`
}

// ProductionStackStatus defines the observed state of ProductionStack.
type ProductionStackStatus struct {
	// AvailableReplicas is the number of available replicas
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Conditions represents the latest available observations of the cluster's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ModelStatus represents the status of the model loading
	ModelStatus string `json:"modelStatus,omitempty"`

	// LastUpdated is the last time the status was updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ps

// ProductionStack is the Schema for the productionstacks API.
type ProductionStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProductionStackSpec   `json:"spec,omitempty"`
	Status ProductionStackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProductionStackList contains a list of ProductionStack.
type ProductionStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProductionStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProductionStack{}, &ProductionStackList{})
}
