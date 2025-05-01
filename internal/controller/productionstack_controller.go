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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	servingv1alpha1 "production-stack/api/v1alpha1"
)

// ProductionStackReconciler reconciles a ProductionStack object
type ProductionStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=serving.vllm.ai,resources=productionstacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.vllm.ai,resources=productionstacks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=serving.vllm.ai,resources=productionstacks/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProductionStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the ProductionStack instance
	productionStack := &servingv1alpha1.ProductionStack{}
	err := r.Get(ctx, req.NamespacedName, productionStack)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("ProductionStack resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ProductionStack")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: productionStack.Name, Namespace: productionStack.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForProductionStack(productionStack)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Update the deployment if needed
	if r.deploymentNeedsUpdate(found, productionStack) {
		log.Info("Updating Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		// Create new deployment spec
		newDep := r.deploymentForProductionStack(productionStack)

		err = r.Update(ctx, newDep)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Deployment updated successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the status
	if err := r.updateStatus(ctx, productionStack, found); err != nil {
		log.Error(err, "Failed to update ProductionStack status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deploymentForProductionStack returns a ProductionStack Deployment object
func (r *ProductionStackReconciler) deploymentForProductionStack(ps *servingv1alpha1.ProductionStack) *appsv1.Deployment {
	labels := map[string]string{
		"app": ps.Name,
	}

	// Build command line arguments
	args := []string{
		"--model",
		ps.Spec.Model.ModelURL,
		"--host",
		"0.0.0.0",
		"--port",
		"8000",
	}

	if ps.Spec.Model.EnableLoRA {
		args = append(args, "--enable-lora")
	}

	if ps.Spec.Model.EnableTool {
		args = append(args, "--enable-auto-tool-choice")
	}

	if ps.Spec.Model.ToolCallParser != "" {
		args = append(args, "--tool-call-parser", ps.Spec.Model.ToolCallParser)
	}

	args = append(args, "--enable-chunked-prefill", fmt.Sprintf("%t", ps.Spec.VLLMConfig.EnableChunkedPrefill))

	if ps.Spec.VLLMConfig.EnablePrefixCaching {
		args = append(args, "--enable-prefix-caching")
	} else {
		args = append(args, "--no-enable-prefix-caching")
	}

	if ps.Spec.Model.MaxModelLen > 0 {
		args = append(args, "--max-model-len", fmt.Sprintf("%d", ps.Spec.Model.MaxModelLen))
	}

	if ps.Spec.Model.DType != "" {
		args = append(args, "--dtype", ps.Spec.Model.DType)
	}

	if ps.Spec.VLLMConfig.TensorParallelSize > 0 {
		args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", ps.Spec.VLLMConfig.TensorParallelSize))
	}

	if ps.Spec.Model.MaxNumSeqs > 0 {
		args = append(args, "--max-num-seqs", fmt.Sprintf("%d", ps.Spec.Model.MaxNumSeqs))
	}

	if ps.Spec.VLLMConfig.GpuMemoryUtilization != "" {
		args = append(args, "--gpu_memory_utilization", ps.Spec.VLLMConfig.GpuMemoryUtilization)
	}

	if ps.Spec.VLLMConfig.MaxLoras > 0 {
		args = append(args, "--max_loras", fmt.Sprintf("%d", ps.Spec.VLLMConfig.MaxLoras))
	}

	if ps.Spec.VLLMConfig.ExtraArgs != nil {
		args = append(args, ps.Spec.VLLMConfig.ExtraArgs...)
	}

	// Build environment variables
	env := []corev1.EnvVar{
		{
			Name:  "HF_HOME",
			Value: "/data",
		},
	}

	if ps.Spec.VLLMConfig.V1 {
		env = append(env, corev1.EnvVar{
			Name:  "VLLM_USE_V1",
			Value: "1",
		})
	}

	if ps.Spec.LMCacheConfig.Enabled {
		env = append(env,
			corev1.EnvVar{
				Name:  "LMCACHE_LOG_LEVEL",
				Value: "DEBUG",
			},
			corev1.EnvVar{
				Name:  "LMCACHE_USE_EXPERIMENTAL",
				Value: "True",
			},
			corev1.EnvVar{
				Name:  "VLLM_RPC_TIMEOUT",
				Value: "1000000",
			},
		)

		if ps.Spec.LMCacheConfig.CPUOffloadingBufferSize != "" {
			env = append(env,
				corev1.EnvVar{
					Name:  "LMCACHE_LOCAL_CPU",
					Value: "True",
				},
				corev1.EnvVar{
					Name:  "LMCACHE_MAX_LOCAL_CPU_SIZE",
					Value: ps.Spec.LMCacheConfig.CPUOffloadingBufferSize,
				},
			)
		}

		if ps.Spec.LMCacheConfig.DiskOffloadingBufferSize != "" {
			env = append(env,
				corev1.EnvVar{
					Name:  "LMCACHE_LOCAL_DISK",
					Value: "True",
				},
				corev1.EnvVar{
					Name:  "LMCACHE_MAX_LOCAL_DISK_SIZE",
					Value: ps.Spec.LMCacheConfig.DiskOffloadingBufferSize,
				},
			)
		}

		if ps.Spec.LMCacheConfig.RemoteURL != "" {
			env = append(env,
				corev1.EnvVar{
					Name:  "LMCACHE_REMOTE_URL",
					Value: ps.Spec.LMCacheConfig.RemoteURL,
				},
				corev1.EnvVar{
					Name:  "LMCACHE_REMOTE_SERDE",
					Value: ps.Spec.LMCacheConfig.RemoteSerde,
				},
			)
		}
	}

	// Add user-defined environment variables
	if ps.Spec.Env != nil {
		for _, e := range ps.Spec.Env {
			env = append(env, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if ps.Spec.Resources.CPU != "" {
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(ps.Spec.Resources.CPU)
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(ps.Spec.Resources.CPU)
	}

	if ps.Spec.Resources.Memory != "" {
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(ps.Spec.Resources.Memory)
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(ps.Spec.Resources.Memory)
	}

	if ps.Spec.Resources.GPU != "" {
		// Parse GPU resource as a decimal value
		gpuResource := resource.MustParse(ps.Spec.Resources.GPU)
		resources.Requests["nvidia.com/gpu"] = gpuResource
		resources.Limits["nvidia.com/gpu"] = gpuResource
	}

	// Get the image from VLLMConfig or use default
	image := ps.Spec.VLLMConfig.Image.Registry + "/" + ps.Spec.VLLMConfig.Image.Name

	// Get the image pull policy
	imagePullPolicy := corev1.PullIfNotPresent
	if ps.Spec.VLLMConfig.Image.PullPolicy != "" {
		imagePullPolicy = corev1.PullPolicy(ps.Spec.VLLMConfig.Image.PullPolicy)
	}

	// Build image pull secrets
	var imagePullSecrets []corev1.LocalObjectReference
	if ps.Spec.VLLMConfig.Image.PullSecretName != "" {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: ps.Spec.VLLMConfig.Image.PullSecretName,
		})
	}

	if ps.Spec.VLLMConfig.HFTokenSecret.Name != "" {
		env = append(env, corev1.EnvVar{
			Name: "HF_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: ps.Spec.VLLMConfig.HFTokenSecret,
					Key:                  "token",
				},
			},
		})
	}

	var strategy appsv1.DeploymentStrategy
	if ps.Spec.DeploymentStrategy == "Recreate" {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	} else {
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		}
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ps.Spec.Replicas,
			Strategy: strategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: imagePullSecrets,
					Containers: []corev1.Container{
						{
							Name:            "vllm",
							Image:           image,
							ImagePullPolicy: imagePullPolicy,
							Command:         []string{"python3", "-m", "vllm.entrypoints.api_server"},
							Args:            args,
							Env:             env,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8000,
								},
							},
							Resources: resources,
						},
					},
				},
			},
		},
	}

	// Set the owner reference
	ctrl.SetControllerReference(ps, dep, r.Scheme)
	return dep
}

// deploymentNeedsUpdate checks if the deployment needs to be updated
func (r *ProductionStackReconciler) deploymentNeedsUpdate(dep *appsv1.Deployment, ps *servingv1alpha1.ProductionStack) bool {
	log := log.FromContext(context.Background())
	// Generate the expected deployment
	expectedDep := r.deploymentForProductionStack(ps)

	// Compare model URL
	expectedModelURL := ps.Spec.Model.ModelURL
	actualModelURL := ""
	// For vllm serve, the model URL is the first argument after the command
	if len(dep.Spec.Template.Spec.Containers[0].Args) > 0 {
		actualModelURL = dep.Spec.Template.Spec.Containers[0].Args[1]
	}
	if expectedModelURL != actualModelURL {
		log.Info("Model URL changed", "expected", expectedModelURL, "actual", actualModelURL)
		return true
	}

	// Compare port
	expectedPort := expectedDep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort
	actualPort := dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort
	if expectedPort != actualPort {
		log.Info("Port changed", "expected", expectedPort, "actual", actualPort)
		return true
	}

	// Compare image
	if expectedDep.Spec.Template.Spec.Containers[0].Image != dep.Spec.Template.Spec.Containers[0].Image {
		log.Info("Image changed",
			"expected", expectedDep.Spec.Template.Spec.Containers[0].Image,
			"actual", dep.Spec.Template.Spec.Containers[0].Image)
		return true
	}

	// Compare LMCache config
	expectedLMCacheEnv := make(map[string]string)
	actualLMCacheEnv := make(map[string]string)
	for _, env := range expectedDep.Spec.Template.Spec.Containers[0].Env {
		if strings.HasPrefix(env.Name, "LMCACHE_") {
			expectedLMCacheEnv[env.Name] = env.Value
		}
	}
	for _, env := range dep.Spec.Template.Spec.Containers[0].Env {
		if strings.HasPrefix(env.Name, "LMCACHE_") {
			actualLMCacheEnv[env.Name] = env.Value
		}
	}
	if !reflect.DeepEqual(expectedLMCacheEnv, actualLMCacheEnv) {
		log.Info("LMCache config changed",
			"expected", expectedLMCacheEnv,
			"actual", actualLMCacheEnv)
		return true
	}

	// Compare resources
	expectedResources := expectedDep.Spec.Template.Spec.Containers[0].Resources
	actualResources := dep.Spec.Template.Spec.Containers[0].Resources
	if !reflect.DeepEqual(expectedResources, actualResources) {
		log.Info("Resources changed",
			"expected", expectedResources,
			"actual", actualResources)
		return true
	}

	return false
}

// updateStatus updates the status of the ProductionStack
func (r *ProductionStackReconciler) updateStatus(ctx context.Context, ps *servingv1alpha1.ProductionStack, dep *appsv1.Deployment) error {
	// Re-read the ProductionStack to get the latest version
	latestPS := &servingv1alpha1.ProductionStack{}
	if err := r.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, latestPS); err != nil {
		return err
	}

	latestPS.Status.AvailableReplicas = dep.Status.AvailableReplicas
	latestPS.Status.LastUpdated = metav1.Now()

	// Update model status based on deployment status
	if dep.Status.AvailableReplicas == latestPS.Spec.Replicas {
		latestPS.Status.ModelStatus = "Ready"
	} else if dep.Status.AvailableReplicas > 0 {
		latestPS.Status.ModelStatus = "PartiallyReady"
	} else if dep.Status.UpdatedReplicas > 0 {
		// If we have updated replicas but they're not yet available, mark as updating
		latestPS.Status.ModelStatus = "Updating"
	} else {
		latestPS.Status.ModelStatus = "NotReady"
	}

	return r.Status().Update(ctx, latestPS)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProductionStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servingv1alpha1.ProductionStack{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
