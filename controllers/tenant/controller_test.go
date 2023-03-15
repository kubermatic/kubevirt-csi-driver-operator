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
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testTimeout = 5 * time.Second
var testInterval = 500 * time.Millisecond

var _ = Describe("Reconcile controller", Ordered, func() {

	BeforeAll(func() {
		startTestEnvWithReconciler()
		createNamespace()
	})
	AfterAll(stopTestEnv)

	Context("CREATE - Tenant object is added", func() {
		BeforeAll(func() {
			testTenant := newTestTenant(testTenantName)
			createTestTenant(testTenant)
			Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: testTenantName}, testTenant)).To(Succeed())
		})

		AfterAll(func() {
			cleanupTenants()
			cleanupOwnedResources()
		})

		It("create should be a success", func() {
			By("checking Daemonset created")
			expectDaemonSetIsCreated()

			By("checking CSIDriver created")
			expectCSIDriverIsCreated()

			By("checking 0 StorageClasses are created")
			expectStorageClassIsNeverCreated()

			By("checking RBAC created")
			expectRbacIsCreated()
		})
	})

	Context("CREATE - Tenant object is added with StorageClasses", func() {
		var tenantStorageClasses []v1alpha1.StorageClass
		var testTenant *v1alpha1.Tenant
		BeforeAll(func() {
			testTenant = newTestTenant(testTenantName)
			tenantStorageClasses = []v1alpha1.StorageClass{
				{InfraStorageClassName: "infra-1", Bus: "scsi"},
				{InfraStorageClassName: "infra-2", Bus: "scsi", IsDefaultClass: pointer.Bool(true)}}
			testTenant.Spec.StorageClasses = tenantStorageClasses
			createTestTenant(testTenant)

			Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: testTenantName}, testTenant)).To(Succeed())
		})

		AfterAll(func() {
			cleanupTenants()
			cleanupOwnedResources()
		})

		It("create should be a success", func() {
			By("checking Daemonset created")
			expectDaemonSetIsCreated()

			By("checking CSIDriver created")
			expectCSIDriverIsCreated()

			By("checking 2 StorageClasses are created")
			expectStorageClassIsCreated(testTenant)

			By("checking RBAC created")
			expectRbacIsCreated()
		})
	})

	Context("UPDATE - Tenant", Ordered, func() {
		var err error
		//var tenantStorageClasses []v1alpha1.StorageClass
		var testTenant *v1alpha1.Tenant
		BeforeAll(func() {
			By("create testTenant")
			testTenant = newTestTenant(testTenantName)
			testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{
				{InfraStorageClassName: "infra-1", Bus: "scsi"},
				{InfraStorageClassName: "infra-2", Bus: "scsi", IsDefaultClass: pointer.Bool(true)}}
			createTestTenant(testTenant)

			Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: testTenantName}, testTenant)).To(Succeed())
			By("checking 2 StorageClasses created ")
			expectStorageClassIsCreated(testTenant)
		})

		AfterAll(func() {
			By("cleanup tenants")
			cleanupTenants()
			cleanupOwnedResources()
		})

		It("update should be a success", func() {
			// Update tenant
			testTenant.Spec.StorageClasses = append(testTenant.Spec.StorageClasses,
				v1alpha1.StorageClass{InfraStorageClassName: "infra-3", Bus: "scsi"})
			Expect(k8sClient.Update(testCtx, testTenant)).To(Succeed())

			err = k8sClient.Get(testCtx, types.NamespacedName{Name: testTenantName}, testTenant)
			Expect(err).To(BeNil())
			Expect(len(testTenant.Spec.StorageClasses)).To(Equal(3))

			By("checking 3 StorageClasses")
			expectStorageClassIsCreated(testTenant)
		})

	})

	Context("CREATE - Tenant otherTenant is added", func() {
		BeforeAll(func() {
			testTenant := newTestTenant(otherTestTenantName)
			createTestTenant(testTenant)
			Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: otherTestTenantName}, testTenant)).To(Succeed())
		})

		AfterAll(func() {
			cleanupTenants()
			cleanupOwnedResources()
		})

		It("create should be a noop", func() {
			By("Daemonset not created")
			expectDaemonSetIsNeverCreated()

			By("CSIDriver not created")
			expectCSIDriverIsNeverCreated()

			By("0 StorageClass created")
			expectStorageClassIsNeverCreated()

			By("RBAC not create")
			expectRbacIsNeverCreated()
		})
	})

})

func expectDaemonSetIsCreated() {
	GinkgoHelper()

	By("expectDaemonSetIsCreated")
	// We'll need to retry getting this newly created object, given that creation may not immediately happen.
	ds := &appsv1.DaemonSet{}
	Eventually(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name:      csiDaemonSetName,
			Namespace: namespaceName}, ds)
		if err != nil {
			return false
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func expectDaemonSetIsNeverCreated() {
	GinkgoHelper()

	By("expectDaemonSetIsNeverCreated")
	ds := &appsv1.DaemonSet{}
	Consistently(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name:      csiDaemonSetName,
			Namespace: namespaceName}, ds)
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func expectCSIDriverIsCreated() {
	GinkgoHelper()

	By("expectCSIDriverIsCreated")
	c := &storagev1.CSIDriver{}
	// We'll need to retry getting this newly created object, given that creation may not immediately happen.
	Eventually(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDriverName}, c)
		if err != nil {
			return false
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func expectCSIDriverIsNeverCreated() {
	GinkgoHelper()

	By("expectCSIDriverIsNeverCreated")
	c := &storagev1.CSIDriver{}
	Consistently(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDriverName}, c)
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func expectStorageClassIsNeverCreated() {
	GinkgoHelper()

	By("expectStorageClassIsNeverCreated")
	Consistently(func() bool {
		sc := &storagev1.StorageClassList{}
		err := k8sClient.List(testCtx, sc)
		if err != nil {
			return false
		}
		return Expect(len(sc.Items)).To(Equal(0))
	})
}

func expectStorageClassIsCreated(tenant *v1alpha1.Tenant) {
	GinkgoHelper()

	By(fmt.Sprintf("expectStorageClassIsCreated [%d]", len(tenant.Spec.StorageClasses)))
	Eventually(func() bool {
		scl := &storagev1.StorageClassList{}
		err := k8sClient.List(testCtx, scl)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("\n... scl size [%d]", len(scl.Items)))

		if len(scl.Items) != len(tenant.Spec.StorageClasses) {
			return false
		}
		for _, tsc := range tenant.Spec.StorageClasses {
			// Should contain the reconciled spec from Tenant Spec.
			opt := cmp.Comparer(cmpStorageClasses)
			if !Expect(scl.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(tenant, tsc), opt))) == true {
				return false
			}
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

}

func expectRbacIsCreated() {
	GinkgoHelper()

	By("expectRbacIsCreated")
	cr := &rbacv1.ClusterRole{}
	Eventually(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDaemonSetName}, cr)
		if err != nil {
			return false
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

	sa := &corev1.ServiceAccount{}
	Eventually(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name:      csiDaemonSetName,
			Namespace: namespaceName}, sa)
		if err != nil {
			return false
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

	crb := &rbacv1.ClusterRoleBinding{}
	Eventually(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDaemonSetName}, crb)
		if err != nil {
			return false
		}
		return true
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func expectRbacIsNeverCreated() {
	GinkgoHelper()

	By("expectRbacIsNeverCreated")
	cr := &rbacv1.ClusterRole{}
	Consistently(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDaemonSetName}, cr)
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

	sa := &corev1.ServiceAccount{}
	Consistently(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name:      csiDaemonSetName,
			Namespace: namespaceName}, sa)
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

	crb := &rbacv1.ClusterRoleBinding{}
	Consistently(func() bool {
		err := k8sClient.Get(testCtx, types.NamespacedName{
			Name: csiDaemonSetName}, crb)
		if err != nil && kerrors.IsNotFound(err) {
			return true
		}
		return false
	}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
}

func cleanupOwnedResources() {
	GinkgoHelper()

	By("cleanupOwnedResources")
	cleanupDaemonSet()
	cleanupCSIDriver()
	cleanupRbac()
	cleanupStorageClasses()
}

func cleanupDaemonSet() {
	GinkgoHelper()

	By("Cleanup Daemonset")
	l := &appsv1.DaemonSetList{}
	Expect(k8sClient.List(testCtx, l)).To(Succeed())
	for _, i := range l.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}
}

func cleanupCSIDriver() {
	GinkgoHelper()

	By("Cleanup CSIDriver")
	l := &storagev1.CSIDriverList{}
	Expect(k8sClient.List(testCtx, l)).To(Succeed())
	for _, i := range l.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}
}

func cleanupRbac() {
	GinkgoHelper()

	By("Cleanup RBAC")
	crl := &rbacv1.ClusterRoleList{}
	Expect(k8sClient.List(testCtx, crl)).To(Succeed())
	for _, i := range crl.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}

	sal := &corev1.ServiceAccountList{}
	Expect(k8sClient.List(testCtx, sal, client.InNamespace(namespaceName))).To(Succeed())
	for _, i := range sal.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}

	crbl := &rbacv1.ClusterRoleBindingList{}
	Expect(k8sClient.List(testCtx, crbl)).To(Succeed())
	for _, i := range crbl.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}
}
