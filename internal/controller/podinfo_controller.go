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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	angiv1 "ryan.evans.com/angi/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// PodInfoReconciler reconciles a PodInfo object
type PodInfoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=angi.ryan.evans.com,resources=podinfoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=angi.ryan.evans.com,resources=podinfoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=angi.ryan.evans.com,resources=podinfoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodInfo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *PodInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	loggr := log.FromContext(ctx)

	loggr.Info("Reconciling")

	podInfoCR := angiv1.PodInfo{}
	err := r.Client.Get(ctx, req.NamespacedName, &podInfoCR)
	if err != nil {
		// We do not care if the object is not found. This will catch
		// the auto-requeue on removing the finalizer
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	CRFinalizer := "angi.ryan.evans.com/finalizer"

	if podInfoCR.DeletionTimestamp.IsZero() {
		// New CR created.
		if !controllerutil.ContainsFinalizer(&podInfoCR, CRFinalizer) {
			// Add a finalizer to allow reconciliation to happen before apiserver deletes the CR
			controllerutil.AddFinalizer(&podInfoCR, CRFinalizer)
			// Update the CR to add the finalizer
			if err := r.Update(ctx, &podInfoCR); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&podInfoCR, CRFinalizer) {
			// CR has a finalizer so we can delete our pod-info deployment now
			if err := r.deletePodInfoDeployment(ctx, podInfoCR); err != nil {
				// If we fail, return the error so the request is requeued.
				return ctrl.Result{}, err
			}
			loggr.Info("Deployment successfully deleted", "pod_info_cr_name", podInfoCR.Name)

			// Remove our finalizer from the CR and update it.
			controllerutil.RemoveFinalizer(&podInfoCR, CRFinalizer)
			if err := r.Update(ctx, &podInfoCR); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Sync the pod-info deployment with the CR
	if err := r.createOrUpdatePodInfoDeployment(ctx, podInfoCR); err != nil {
		return ctrl.Result{}, err
	}
	loggr.Info("Deployment successfully created / updated", "pod_info_cr_name", podInfoCR.Name)

	return ctrl.Result{}, nil
}

// deletePodInfoDeployment will delete the PodInfo deployment for a given PodInfo CR
func (r *PodInfoReconciler) deletePodInfoDeployment(ctx context.Context, podInfoCR angiv1.PodInfo) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podInfoCR.Name,
			Namespace: "default",
		},
	}

	err := r.Delete(ctx, deployment)
	if err != nil {
		return err
	}
	return nil
}

// createOrUpdatePodInfoDeployment creates or updates a deployment K8s resource based on a given
// PodInfo CR
func (r *PodInfoReconciler) createOrUpdatePodInfoDeployment(ctx context.Context, podInfoCR angiv1.PodInfo) error {

	// Convert the UI settings from the CR into a byte slice
	envVars, err := yaml.Marshal(podInfoCR.Spec.UI)
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podInfoCR.Name,
			Namespace: "default",
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(podInfoCR.Spec.ReplicaCount)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "podinfo",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: intstrPtr(0),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// We hash the env vars (the UI of the spec of the CR) to ensure the pods
					// roll whenever the env vars chagne. This allows us to hotdeploy var changes
					Annotations: map[string]string{
						"envVars": sha256Encode(string(envVars)),
					},
					Labels: map[string]string{
						"app": "podinfo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podInfoCR.Name,
							Image: fmt.Sprintf("%s:%s", podInfoCR.Spec.Image.Repository, podInfoCR.Spec.Image.Tag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9898,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PODINFO_UI_COLOR",
									Value: podInfoCR.Spec.UI.Color,
								},
								{
									Name:  "PODINFO_UI_MESSAGE",
									Value: podInfoCR.Spec.UI.Message,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(podInfoCR.Spec.Resources.MemoryLimit),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse(podInfoCR.Spec.Resources.CpuRequest),
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// int32Ptr will convert an int32 to a pointer to an int32
func int32Ptr(i int32) *int32 { return &i }

// intstrPtr will convert an int to a pointer to an intstr
func intstrPtr(value int) *intstr.IntOrString {
	intValue := intstr.FromInt(value)
	return &intValue
}

// sha256Encode will hash a given string and return the hash
func sha256Encode(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&angiv1.PodInfo{}).
		Complete(r)
}
