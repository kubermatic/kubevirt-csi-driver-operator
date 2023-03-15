/*
Copyright 2022 The KubeVirt CSI driver Operator Authors.

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
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconcile storageClass", Ordered, func() {

	var testReconcile *TenantReconciler
	var createOrUpdateError bool
	var err error
	var storageClasses []client.Object
	var tenantStorageClass1 v1alpha1.StorageClass
	var tenantStorageClass2 v1alpha1.StorageClass

	var testTenant *v1alpha1.Tenant
	var otherTestTenant *v1alpha1.Tenant

	BeforeAll(func() {
		tenantStorageClass1 = v1alpha1.StorageClass{InfraStorageClassName: "infra-1", Bus: "scsi"}
		tenantStorageClass2 = v1alpha1.StorageClass{InfraStorageClassName: "infra-2", Bus: "scsi"}

		startTestEnv()
	})

	JustBeforeEach(func() {
		By("Create user cluster StorageClasses")
		createStorageClasses(storageClasses)

		By("Initiliazing Reconcile")
		testReconcile = &TenantReconciler{}
		testReconcile.Client = &fakeClientWithError{
			Client:        k8sClient,
			generateError: createOrUpdateError,
		}

		By("Reconcile")
		err = testReconcile.reconcileStorageClasses(testCtx, testTenant.GetObjectMeta(), testTenant.Spec.StorageClasses)
	})

	AfterEach(func() {
		cleanupTenants()
		cleanupStorageClasses()
		testTenant = nil
		otherTestTenant = nil
		storageClasses = []client.Object{}
	})

	AfterAll(stopTestEnv)

	Context("When Tenant has 1 StorageClass without optional fields", func() {
		scList := v1.StorageClassList{}

		BeforeEach(func() {
			By("New Tenant")
			testTenant = newTestTenant(testTenantName)
			otherTestTenant = newTestTenant(otherTestTenantName)
			By("Create Tenant")
			testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tenantStorageClass1}
			createTestTenant(testTenant)
			createTestTenant(otherTestTenant)
		})

		JustBeforeEach(func() {
			Expect(k8sClient.List(testCtx, &scList)).To(Succeed())
		})

		Context("CREATE - When User cluster has 0 StorageClass", func() {
			BeforeEach(func() {
				By("Init User cluster")
				storageClasses = []client.Object{}
			})
			It("should create the StorageClass", func() {
				Expect(len(scList.Items)).Should(Equal(1))
				opt := cmp.Comparer(cmpStorageClasses)
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
			})
		})

		Context("NOOP - When User cluster has 1 StorageClass that is in the Tenant list, with correct Spec, owned by the managed Tenant", func() {
			BeforeEach(func() {
				By("Init User cluster")
				userSc := newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1)
				storageClasses = []client.Object{&userSc}
			})

			It("should be a noop", func() {
				Expect(len(scList.Items)).Should(Equal(1))
				opt := cmp.Comparer(cmpStorageClasses)
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
			})
		})

		Context("UPDATE FORBIDDEN - When User cluster has 1 StorageClass that is in the Tenant list, with Spec to update, owned by the managed Tenant", func() {
			BeforeEach(func() {
				By("Init User cluster")
				userSc := newStorageClassFromTenantWithDiff(testTenant, tenantStorageClass1)
				storageClasses = []client.Object{&userSc}
			})

			It("should not the StorageClass to match the tenant Spec and return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("UPDATE ALLOWED - When User cluster has 1 StorageClass that is in the Tenant list, with Spec to update - only isDefaultClass, owned by the managed Tenant", func() {
			BeforeEach(func() {
				By("Init User cluster")
				userSc := newStorageClassFromTenantWithOnlyIsDefaultClassDiff(testTenant, tenantStorageClass1)
				storageClasses = []client.Object{&userSc}
			})

			It("should update the isDefaultClass in the StorageClass to match the tenant Spec", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(len(scList.Items)).Should(Equal(1))
				opt := cmp.Comparer(cmpStorageClasses)
				// Shoult Not contain the "withdiff" version
				Expect(scList.Items).ShouldNot(ContainElement(BeComparableTo(newStorageClassFromTenantWithOnlyIsDefaultClassDiff(testTenant, tenantStorageClass1), opt)))
				// But should contain the reconciled spec from Tenant Spec.
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
			})
		})

		// This test will need to be modified when we implement: https://github.com/kubermatic/kubermatic/issues/11998
		// For now, we do not delete extra Storage Classes in the user cluster that are not part of the Tenant (with same ownerReferences).
		// After we implement it, we should delete the StorageClass userSc0 that has the same ownerRefernces but is not in the Tenant spec.
		Context("When User cluster has 1 StorageClass that is NOT in the Tenant list, owned by the managed Tenant", func() {
			BeforeEach(func() {
				By("Init User cluster")
				userSc := newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass2)
				storageClasses = []client.Object{&userSc}
			})

			It("should create the StorageClass from Tenant, and not delete the extra one", func() {
				Expect(len(scList.Items)).Should(Equal(2))
				opt := cmp.Comparer(cmpStorageClasses)
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass2), opt)))

			})
		})

		Context("hen User cluster has 1 StorageClass that is NOT in the Tenant list, owned by another Tenant", func() {
			BeforeEach(func() {
				By("Init User cluster")
				userSc := newStorageClassFromTenantNoDiff(otherTestTenant, tenantStorageClass2)
				storageClasses = []client.Object{&userSc}
			})

			It("should not delete the StorageClass owned by the other Tenant", func() {
				Expect(err).NotTo(HaveOccurred())
				scList := v1.StorageClassList{}
				Expect(k8sClient.List(testCtx, &scList)).NotTo(HaveOccurred())

				Expect(len(scList.Items)).Should(Equal(2))
				opt := cmp.Comparer(cmpStorageClasses)
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(otherTestTenant, tenantStorageClass2), opt)))

			})
		})

	})

	Context("OPTIONAL FIELDS - When Tenant has 1 StorageClass with optional fields", func() {
		scList := v1.StorageClassList{}

		BeforeEach(func() {
			storageClasses = []client.Object{}
		})

		JustBeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.List(testCtx, &scList)).NotTo(HaveOccurred())
			Expect(len(scList.Items)).Should(Equal(1))
		})

		Context("Field isDefault=true", func() {
			BeforeEach(func() {
				By("New Tenant")
				testTenant = newTestTenant(testTenantName)
				otherTestTenant = newTestTenant(otherTestTenantName)
				By("Create Tenant")
				tsc := v1alpha1.StorageClass{InfraStorageClassName: "infra-1", IsDefaultClass: pointer.Bool(true), Bus: "scsi"}
				testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tsc}
				createTestTenant(testTenant)
				createTestTenant(otherTestTenant)

			})

			It("should have an annotation isDefault=true ", func() {
				// "infraStorageClassName" key should be present
				Expect(isDefaultStorageClassannotationKey).Should(BeKeyOf(scList.Items[0].Annotations))
				Expect(scList.Items[0].Annotations[isDefaultStorageClassannotationKey]).Should(Equal("true"))
			})
		})

		Context("Field isDefault=false", func() {
			BeforeEach(func() {
				By("New Tenant")
				testTenant = newTestTenant(testTenantName)
				otherTestTenant = newTestTenant(otherTestTenantName)
				By("Create Tenant")
				tsc := v1alpha1.StorageClass{InfraStorageClassName: "infra-1", IsDefaultClass: pointer.Bool(false), Bus: "scsi"}
				testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tsc}
				createTestTenant(testTenant)
				createTestTenant(otherTestTenant)
			})
			It("should have an annotation isDefault=false ", func() {
				// "infraStorageClassName" key should be present
				Expect(isDefaultStorageClassannotationKey).Should(BeKeyOf(scList.Items[0].Annotations))
				Expect(scList.Items[0].Annotations[isDefaultStorageClassannotationKey]).Should(Equal("false"))
			})
		})

		Context("Field isDefault not present", func() {
			BeforeEach(func() {
				By("New Tenant")
				testTenant = newTestTenant(testTenantName)
				otherTestTenant = newTestTenant(otherTestTenantName)
				By("Create Tenant")
				tsc := v1alpha1.StorageClass{InfraStorageClassName: "infra-1", Bus: "scsi"}
				testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tsc}
				createTestTenant(testTenant)
				createTestTenant(otherTestTenant)
			})
			It("should have an annotation isDefault=false ", func() {
				// "infraStorageClassName" key should be present
				Expect(isDefaultStorageClassannotationKey).Should(BeKeyOf(scList.Items[0].Annotations))
				Expect(scList.Items[0].Annotations[isDefaultStorageClassannotationKey]).Should(Equal("false"))
			})
		})

		Context("Field Bus not present", func() {
			BeforeEach(func() {
				By("New Tenant")
				testTenant = newTestTenant(testTenantName)
				otherTestTenant = newTestTenant(otherTestTenantName)
				By("Create Tenant")
				tsc := v1alpha1.StorageClass{InfraStorageClassName: "infra-1"}
				testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tsc}
				createTestTenant(testTenant)
				createTestTenant(otherTestTenant)
			})
			It("should not have the bus parameter in the StorageClass", func() {
				Expect(err).NotTo(HaveOccurred())
				scList := v1.StorageClassList{}
				Expect(k8sClient.List(testCtx, &scList)).NotTo(HaveOccurred())

				Expect(len(scList.Items)).Should(Equal(1))
				// "bus" key should not be present
				Expect("bus").ShouldNot(BeKeyOf(scList.Items[0].Parameters))
			})
		})

	})

	Context("When managed Tenant has 2 StorageClass", func() {
		BeforeEach(func() {
			By("New Tenant")
			testTenant = newTestTenant(testTenantName)
			otherTestTenant = newTestTenant(otherTestTenantName)
			By("Create Tenant")
			testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tenantStorageClass1, tenantStorageClass2}
			createTestTenant(testTenant)
			createTestTenant(otherTestTenant)
		})

		Context("When User cluster has the 2 StorageClasses that are in the Tenant list, with 1 Spec to update, 1 Spec with noop, owned by the managed Tenant", func() {
			scList := v1.StorageClassList{}

			BeforeEach(func() {
				By("Init User cluster")
				userSc1 := newStorageClassFromTenantWithOnlyIsDefaultClassDiff(testTenant, tenantStorageClass1)
				userSc2 := newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass2)
				storageClasses = []client.Object{&userSc1, &userSc2}
			})

			JustBeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.List(testCtx, &scList)).NotTo(HaveOccurred())
			})

			It("should be update the StorageClass to match the tenant Spec for sc1 and noop for sc2", func() {

				Expect(len(scList.Items)).Should(Equal(2))
				opt := cmp.Comparer(cmpStorageClasses)
				// Shoult Not contain the "withdiff" version
				Expect(scList.Items).ShouldNot(ContainElement(BeComparableTo(newStorageClassFromTenantWithOnlyIsDefaultClassDiff(testTenant, tenantStorageClass1), opt)))
				// But should contain the reconciled spec from Tenant Spec.
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass1), opt)))
				// Should contains the noop for sc2
				Expect(scList.Items).Should(ContainElement(BeComparableTo(newStorageClassFromTenantNoDiff(testTenant, tenantStorageClass2), opt)))

			})
		})

	})

	Context("ERROR - When CreateOrUpdate returns an error", func() {
		BeforeEach(func() {
			By("New Tenant")
			testTenant = newTestTenant(testTenantName)
			otherTestTenant = newTestTenant(otherTestTenantName)
			By("Create Tenant")
			testTenant.Spec.StorageClasses = []v1alpha1.StorageClass{tenantStorageClass1, tenantStorageClass2}
			createTestTenant(testTenant)
			createTestTenant(otherTestTenant)

			By("Setting client error response")
			createOrUpdateError = true
			By("Init User cluster StorageClasses")
			storageClasses = []client.Object{}
		})

		It("reconcile should return an error", func() {
			Expect(err).To(HaveOccurred())

		})

		It("should not create StorageClasses", func() {
			scList := v1.StorageClassList{}
			Expect(k8sClient.List(testCtx, &scList)).NotTo(HaveOccurred())
			Expect(len(scList.Items)).Should(Equal(0))
		})

	})

})

type fakeClientWithError struct {
	client.Client
	generateError bool
}

func (fake *fakeClientWithError) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if fake.generateError {
		return errors.New("error creating storageClass")
	}
	return fake.Client.Create(ctx, obj, opts...)
}

// newStorageClassFromTenantNoDiff returns a StorageClass that matches exactly the infra StorageClass defined in the tenant
func newStorageClassFromTenantNoDiff(owner *v1alpha1.Tenant, tenantSc v1alpha1.StorageClass) v1.StorageClass {
	return v1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf(getUserClusterStorageClassName(tenantSc.InfraStorageClassName)),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
			Annotations: map[string]string{
				isDefaultStorageClassannotationKey: strconv.FormatBool(tenantSc.IsDefaultClass != nil && *tenantSc.IsDefaultClass),
			},
		},
		Provisioner: provisioner,
		Parameters: map[string]string{
			"infraStorageClassName": tenantSc.InfraStorageClassName,
			"bus":                   tenantSc.Bus},
	}
}

// newStorageClassFromTenantWithDiff returns a StorageClass that differs from the tenant StorageClass
// but has the same name and same owner.
// This is used to create a StorageClass that needs to be updated by the reconcile loop.
func newStorageClassFromTenantWithDiff(owner *v1alpha1.Tenant, tenantSc v1alpha1.StorageClass) v1.StorageClass {
	isDefault := (tenantSc.IsDefaultClass != nil && !*tenantSc.IsDefaultClass) || tenantSc.IsDefaultClass == nil
	sc := newStorageClassFromTenantNoDiff(owner, tenantSc)
	sc.Annotations = map[string]string{
		isDefaultStorageClassannotationKey: strconv.FormatBool(isDefault)}
	sc.Provisioner = fmt.Sprintf("wrong-%s", provisioner)
	sc.Parameters = map[string]string{
		"infraStorageClassName": fmt.Sprintf("wrong-%s", tenantSc.InfraStorageClassName),
		"bus":                   fmt.Sprintf("wrong-%s", tenantSc.Bus)}
	return sc
}

// newStorageClassFromTenantWithOnlyIsDefaultClassDiff returns a StorageClass that differs from the tenant StorageClass
// but has the same name and same owner.
// This is used to create a StorageClass that needs to be updated by the reconcile loop.
func newStorageClassFromTenantWithOnlyIsDefaultClassDiff(owner *v1alpha1.Tenant, tenantSc v1alpha1.StorageClass) v1.StorageClass {
	isDefault := (tenantSc.IsDefaultClass != nil && !*tenantSc.IsDefaultClass) || tenantSc.IsDefaultClass == nil
	sc := newStorageClassFromTenantNoDiff(owner, tenantSc)
	sc.Annotations = map[string]string{
		isDefaultStorageClassannotationKey: strconv.FormatBool(isDefault)}
	return sc
}

// cmpStorageClasses compares 2 StorageClasses regarding a set of fields
// we want to ensure:
// - Name/Provisioner/Parameters/OwnerReferences/Annotations
func cmpStorageClasses(sc1, sc2 v1.StorageClass) bool {
	return sc1.Name == sc2.Name &&
		sc1.Provisioner == sc2.Provisioner &&
		reflect.DeepEqual(sc1.Annotations, sc2.Annotations) &&
		reflect.DeepEqual(sc1.Parameters, sc2.Parameters) &&
		reflect.DeepEqual(sc1.OwnerReferences, sc2.OwnerReferences)
}

func createStorageClasses(storageClasses []client.Object) {
	GinkgoHelper()

	for _, sc := range storageClasses {
		Expect(k8sClient.Create(testCtx, sc)).To(Succeed())
	}
}

func cleanupStorageClasses() {
	GinkgoHelper()

	By("Clean StorageClasses")
	scList := &v1.StorageClassList{}
	Expect(k8sClient.List(testCtx, scList)).To(Succeed())
	for _, sc := range scList.Items {
		Expect(k8sClient.Delete(testCtx, &sc)).To(Succeed())
	}
}
