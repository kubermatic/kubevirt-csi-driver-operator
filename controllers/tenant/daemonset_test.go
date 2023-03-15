/*
Copyright 2023 The KubeVirt CSI driver Operator Authors.

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

package tenant

import (
	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Reconcile daemonset", Ordered, func() {
	var testReconcile *TenantReconciler
	var createOrUpdateError bool
	imageRepository := ""
	csiDriverTag := ""
	csiSidecarTag := ""
	var csiDriverDefaultTag, csiSidecarDefaultTag string
	var err error
	var testTenant *v1alpha1.Tenant

	BeforeAll(func() {
		startTestEnv()
		createNamespace()

		csiDriverDefaultTag, _ = getTag(csiDriverImage)
		csiSidecarDefaultTag, _ = getTag(csiNodeDriveRegistrarImage)
	})

	AfterAll(stopTestEnv)

	JustBeforeEach(func() {
		testTenant = newTestTenant(testTenantName)
		createTestTenant(testTenant)

		By("Initiliazing Reconcile")
		testReconcile = &TenantReconciler{}
		testReconcile.Client = &fakeClientWithError{
			Client:        k8sClient,
			generateError: createOrUpdateError,
		}
		By("Reconcile")
		_, err = testReconcile.reconcileDaemonset(testCtx, testTenant.GetObjectMeta(), imageRepository, csiDriverTag, csiSidecarTag)
	})

	AfterEach(func() {
		cleanupTenants()
		testTenant = nil

		csiDriverTag = ""
		csiSidecarTag = ""
		imageRepository = ""
	})

	Context("ImageRepository not provided - ImageTag not provided", func() {
		ds := &appsv1.DaemonSet{}

		JustBeforeEach(func() {
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      csiDaemonSetName,
				Namespace: namespaceName}, ds)).NotTo(HaveOccurred())
		})

		It("Should have the default registry for the images", func() {
			Expect(err).NotTo(HaveOccurred())

			By("checking registry")
			expectContainerImageRegistry(ds, "csi-driver", defaultRegistry)
			expectContainerImageRegistry(ds, "csi-node-driver-registrar", defaultRegistry)
			expectContainerImageRegistry(ds, "csi-liveness-probe", defaultRegistry)

			By("checking tag")
			expectContainerImageTag(ds, "csi-driver", csiDriverDefaultTag)
			expectContainerImageTag(ds, "csi-node-driver-registrar", csiSidecarDefaultTag)
			expectContainerImageTag(ds, "csi-liveness-probe", csiSidecarDefaultTag)
		})
	})

	Context("ImageRepository is provided - no ImageTag provided", func() {
		ds := &appsv1.DaemonSet{}

		BeforeEach(func() {
			By("Setting imageRepository")
			imageRepository = "anotherregistry.io"
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      csiDaemonSetName,
				Namespace: namespaceName}, ds)).NotTo(HaveOccurred())
		})
		It("should have the correct registry and tag", func() {
			Expect(err).NotTo(HaveOccurred())

			By("checking registry")
			expectContainerImageRegistry(ds, "csi-driver", imageRepository)
			expectContainerImageRegistry(ds, "csi-node-driver-registrar", imageRepository)
			expectContainerImageRegistry(ds, "csi-liveness-probe", imageRepository)

			By("checking tag")
			expectContainerImageTag(ds, "csi-driver", csiDriverDefaultTag)
			expectContainerImageTag(ds, "csi-node-driver-registrar", csiSidecarDefaultTag)
			expectContainerImageTag(ds, "csi-liveness-probe", csiSidecarDefaultTag)
		})

	})

	Context("ImageRepository not provided -  ImageTag is provided", func() {
		ds := &appsv1.DaemonSet{}

		BeforeEach(func() {
			By("Setting imageTag")
			csiDriverTag = "1.2.3"
			csiSidecarTag = "4.5.6"
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      csiDaemonSetName,
				Namespace: namespaceName}, ds)).To(Succeed())
		})

		It("should have the correct registry and tag", func() {
			Expect(err).NotTo(HaveOccurred())

			By("checking registry")
			expectContainerImageRegistry(ds, "csi-driver", defaultRegistry)
			expectContainerImageRegistry(ds, "csi-node-driver-registrar", defaultRegistry)
			expectContainerImageRegistry(ds, "csi-liveness-probe", defaultRegistry)

			By("checking tag")
			expectContainerImageTag(ds, "csi-driver", csiDriverTag)
			expectContainerImageTag(ds, "csi-node-driver-registrar", csiSidecarTag)
			expectContainerImageTag(ds, "csi-liveness-probe", csiSidecarTag)
		})
	})

	Context("ImageRepository provided -  ImageTag is provided", func() {
		ds := &appsv1.DaemonSet{}

		BeforeEach(func() {
			By("Setting imageTag")
			csiDriverTag = "1.2.3"
			csiSidecarTag = "4.5.6"
			By("Setting imageRepository")
			imageRepository = "anotherregistry.io"
		})

		JustBeforeEach(func() {
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      csiDaemonSetName,
				Namespace: namespaceName}, ds)).To(Succeed())
		})

		It("should have the correct registry and tag", func() {
			Expect(err).NotTo(HaveOccurred())

			By("checking registry")
			expectContainerImageRegistry(ds, "csi-driver", imageRepository)
			expectContainerImageRegistry(ds, "csi-node-driver-registrar", imageRepository)
			expectContainerImageRegistry(ds, "csi-liveness-probe", imageRepository)

			By("checking tag")
			expectContainerImageTag(ds, "csi-driver", csiDriverTag)
			expectContainerImageTag(ds, "csi-node-driver-registrar", csiSidecarTag)
			expectContainerImageTag(ds, "csi-liveness-probe", csiSidecarTag)
		})
	})

})

func expectContainerImageRegistry(ds *appsv1.DaemonSet, containerName, registry string) {
	GinkgoHelper()

	By("checking container image registry")
	r, err := getRegistry(getContainer(ds, containerName).Image)
	Expect(err).NotTo(HaveOccurred())
	Expect(r).Should(Equal(registry))
}

func expectContainerImageTag(ds *appsv1.DaemonSet, containerName, tag string) {
	GinkgoHelper()

	By("checking container image tag")
	t, err := getTag(getContainer(ds, containerName).Image)
	Expect(err).NotTo(HaveOccurred())
	Expect(t).Should(Equal(tag))
}

func getContainer(ds *appsv1.DaemonSet, name string) corev1.Container {
	for _, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c
		}
	}
	return corev1.Container{}
}

func getRegistry(image string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q: %w", image, err)
	}
	return reference.Domain(named), nil
}

func getTag(image string) (string, error) {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q: %w", image, err)
	}
	if tagged, ok := named.(reference.Tagged); ok {
		return tagged.Tag(), nil
	}
	return "", nil
}

func createNamespace() {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
	Expect(k8sClient.Create(testCtx, ns)).To(Succeed())
}
