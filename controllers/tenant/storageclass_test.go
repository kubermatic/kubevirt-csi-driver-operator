package tenant

import (
	"context"
	"errors"

	"github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Reconcile storageClass", func() {
	var testReconcile *TenantReconciler
	var testClient client.Client
	Context("When storageClass is reconciled", func() {
		BeforeEach(func() {
			testClient = fake.NewClientBuilder().Build()
			testReconcile = &TenantReconciler{
				Client: testClient,
			}
		})

		It("should get created", func() {
			testTenant := createTestTenant([]v1alpha1.StorageClass{{InfraStorageClassName: "test-local-path-1", Bus: "scsi"}, {InfraStorageClassName: "test-local-path-2", Bus: "scsi"}})
			Expect(testReconcile.reconcileStorageClasses(context.TODO(), testTenant.GetObjectMeta(), testTenant.Spec.StorageClasses)).NotTo(HaveOccurred())
			scList := v1.StorageClassList{}
			Expect(testClient.List(context.TODO(), &scList)).NotTo(HaveOccurred())
			Expect(len(scList.Items)).Should(Equal(2))
		})

		It("should return an error in case of CreateOrUpdate failure", func() {
			testTenant := createTestTenant([]v1alpha1.StorageClass{{InfraStorageClassName: "test-local-path-1", Bus: "scsi"}})
			testReconcile.Client = &fakeClientWithError{
				Client:        testClient,
				generateError: true,
			}
			err := testReconcile.reconcileStorageClasses(context.TODO(), testTenant.GetObjectMeta(), testTenant.Spec.StorageClasses)
			Expect(err).To(HaveOccurred())
		})

	})

})

func createTestTenant(storageClasses []v1alpha1.StorageClass) *v1alpha1.Tenant {
	return &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant"},
		Spec: v1alpha1.TenantSpec{
			StorageClasses: storageClasses,
		},
	}
}

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
