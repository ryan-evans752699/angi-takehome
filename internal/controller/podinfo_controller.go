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
	r.Client.Get(ctx, req.NamespacedName, &podInfoCR)
	loggr.Info(podInfoCR.Name)

	loggr.Info(podInfoCR.Spec.UI.Message)

	if err := r.reconcilePodInfoConfigmap(ctx, podInfoCR); err != nil {
		loggr.Error(err, "Error recociling configmap for CR", "CR", podInfoCR.Name)

	}

	if err := r.reconcilePodInfoDeployment(ctx, podInfoCR); err != nil {
		loggr.Error(err, "Error recociling deployment for CR", "CR", podInfoCR.Name)

	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func (r *PodInfoReconciler) reconcilePodInfoDeployment(ctx context.Context, podInfo angiv1.PodInfo) error {
	loggr := log.FromContext(ctx)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podInfo.Name,
			Namespace: "default",
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(podInfo.Spec.ReplicaCount)),
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
					Labels: map[string]string{
						"app": "podinfo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podInfo.Name,
							Image: fmt.Sprintf("%s:%s", podInfo.Spec.Image.Repository, podInfo.Spec.Image.Tag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9898,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "PODINFO_UI_COLOR",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: podInfo.Name},
											Key:                  "PODINFO_UI_COLOR",
										},
									},
								},
								{
									Name: "PODINFO_UI_MESSAGE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: podInfo.Name},
											Key:                  "PODINFO_UI_MESSAGE",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(podInfo.Spec.Resources.MemoryLimit),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse(podInfo.Spec.Resources.CpuRequest),
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

	loggr.Info("Deployment reconciled", "operation", op)

	return nil
}

func (r *PodInfoReconciler) reconcilePodInfoConfigmap(ctx context.Context, podInfo angiv1.PodInfo) error {
	loggr := log.FromContext(ctx)

	// Define the ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podInfo.Name,
			Namespace: "default",
		},
	}

	// Perform create or update operation on the ConfigMap
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		// Update the data in the ConfigMap
		configMap.Data = map[string]string{
			"PODINFO_UI_COLOR":   podInfo.Spec.UI.Color,
			"PODINFO_UI_MESSAGE": podInfo.Spec.UI.Message,
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Log the operation result
	loggr.Info("ConfigMap successfully reconciled", "operation", op)

	return nil
}

func int32Ptr(i int32) *int32 { return &i }

func intstrPtr(value int) *intstr.IntOrString {
	intValue := intstr.FromInt(value)
	return &intValue
}

func sha256Encode(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// func (r *PodInfoReconciler) redisSpec() error {
//     pod := &corev1.Pod{
//         ObjectMeta: metav1.ObjectMeta{
//             Name:      "redis-pod",
//             Namespace: "default",
//         },
//         Spec: corev1.PodSpec{
//             Containers: []corev1.Container{
//                 {
//                     Name:  "redis",
//                     Image: "redis",
//                     Ports: []corev1.ContainerPort{
//                         {
//                             ContainerPort: 1234,
//                         },
//                     },
//                 },
//             },
//         },
//     }

//     // Create the Pod
//     _, err := clientset.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
//     if err != nil {
//         return err
//     }

//     return nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *PodInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&angiv1.PodInfo{}).
		Complete(r)
}
