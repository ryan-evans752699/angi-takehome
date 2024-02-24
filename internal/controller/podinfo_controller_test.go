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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	angiv1 "ryan.evans.com/angi/api/v1"
)

var _ = Describe("PodInfo Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName = "test-resource"
			namespace    = "default"
			memoryLimit  = "64Mi"
			cpuRequests  = "100m"
			imageRepo    = "ghcr.io/stefanprodan/podinfo"
			imageTag     = "latests"
			uiColor      = "#34577c"
			uiMessage    = "ryan is awesome 1234 :)"
			uiMessage2   = "I have been modified"
			redisEnabled = true
		)

		ctx := context.Background()

		crNamespaceName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		redisServiceNamespacedName := types.NamespacedName{Name: fmt.Sprintf("%s-redis-service", resourceName), Namespace: "default"}
		redisCMNamespacedName := types.NamespacedName{Name: fmt.Sprintf("%s-redis-cm", resourceName), Namespace: "default"}
		redisDeploymentNamespacedName := types.NamespacedName{Name: fmt.Sprintf("%s-redis-deployment", resourceName), Namespace: "default"}
		podInfoDeploymentNamespacedName := types.NamespacedName{Name: fmt.Sprintf("%s-pod-info-deployment", resourceName), Namespace: "default"}

		podInfo := &angiv1.PodInfo{}

		BeforeEach(func() {
			By("checking there are no Pod Info CRs")
			err := k8sClient.Get(ctx, crNamespaceName, podInfo)
			Expect(errors.IsNotFound(err))
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance PodInfo")
			err := k8sClient.Get(ctx, crNamespaceName, podInfo)
			if !errors.IsNotFound(err) {
				Expect(k8sClient.Delete(ctx, podInfo)).To(Succeed())
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should successfully reconcile the created resource", func() {

			By("creating the custom resource")
			podInfo = getPodInfoCustomResourceDefinition(resourceName, memoryLimit, cpuRequests, imageRepo,
				imageTag, uiColor, uiMessage, redisEnabled)
			err := k8sClient.Create(ctx, podInfo)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling the custom resource")
			controllerReconciler := &PodInfoReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating redis resources")
			redisService := &corev1.Service{}
			err = controllerReconciler.Client.Get(ctx, redisServiceNamespacedName, redisService)
			// Assert that a redis service exists
			Expect(err).NotTo(HaveOccurred())

			redisDeployment := &appsv1.Deployment{}
			err = controllerReconciler.Client.Get(ctx, redisDeploymentNamespacedName, redisDeployment)
			// Assert that a redis deployment exists
			Expect(err).NotTo(HaveOccurred())

			redisCM := &corev1.ConfigMap{}
			err = controllerReconciler.Client.Get(ctx, redisCMNamespacedName, redisCM)
			// Assert that a redis configmap exists
			Expect(err).NotTo(HaveOccurred())

			By("creating podinfo resources")
			podInfoDeployment := &appsv1.Deployment{}
			err = controllerReconciler.Client.Get(ctx, podInfoDeploymentNamespacedName, podInfoDeployment)
			// Assert that a pod info deployment exists
			Expect(err).NotTo(HaveOccurred())

			By("checking finalizer on custom resource")
			// Pull the PodInfo CR after reconcile
			err = controllerReconciler.Client.Get(ctx, crNamespaceName, podInfo)
			Expect(err).NotTo(HaveOccurred())
			// Assert that the finalizer 'CRFinalizer' exists
			Expect(podInfo.Finalizers).To(ContainElement(CRFinalizer))

			By("checking cache value on custom resource")
			// Assert that the cache value on the CR was updated correctly
			Expect(podInfo.Spec.UI.Cache).Should(Equal(fmt.Sprintf("tcp://%s:%d", redisService.Spec.ClusterIP, redisService.Spec.Ports[0].Port)))
		})

		It("should successfully reconcile the created resource", func() {

			By("reconciling the custom resource")
			controllerReconciler := &PodInfoReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource")
			podInfo = getPodInfoCustomResourceDefinition(resourceName, memoryLimit, cpuRequests, imageRepo,
				imageTag, uiColor, uiMessage, redisEnabled)
			err = k8sClient.Create(ctx, podInfo)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("deleting the custom resource")
			// Pull the PodInfo CR after reconcile
			err = controllerReconciler.Client.Get(ctx, crNamespaceName, podInfo)
			Expect(err).NotTo(HaveOccurred())

			// Delete the PodInfo CR
			err = controllerReconciler.Client.Delete(ctx, podInfo)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling the custom resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("should successfully reconcile the modified resource", func() {

			By("creating the custom resource")
			podInfo = getPodInfoCustomResourceDefinition(resourceName, memoryLimit, cpuRequests, imageRepo,
				imageTag, uiColor, uiMessage, redisEnabled)
			err := k8sClient.Create(ctx, podInfo)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling the custom resource")
			controllerReconciler := &PodInfoReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("getting the current PodInfo deployment")
			podInfoDeployment := &appsv1.Deployment{}
			err = controllerReconciler.Client.Get(ctx, podInfoDeploymentNamespacedName, podInfoDeployment)
			// Assert that a pod info deployment exists
			Expect(err).NotTo(HaveOccurred())

			By("modifying the custom resource")
			// Pull the PodInfo CR after reconcile
			err = controllerReconciler.Client.Get(ctx, crNamespaceName, podInfo)
			Expect(err).NotTo(HaveOccurred())
			podInfo.Spec.UI.Message = uiMessage2
			err = k8sClient.Update(ctx, podInfo)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling the custom resource")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: crNamespaceName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the PodInfo deployment was changed")
			updatedPodInfoDeployment := &appsv1.Deployment{}
			err = controllerReconciler.Client.Get(ctx, podInfoDeploymentNamespacedName, updatedPodInfoDeployment)
			// Assert that a pod info deployment exists
			Expect(err).NotTo(HaveOccurred())

			Expect(podInfoDeployment.ResourceVersion != updatedPodInfoDeployment.ResourceVersion)
		})
	})
})
