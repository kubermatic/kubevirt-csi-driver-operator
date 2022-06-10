package tenant

import (
	"context"
	"fmt"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	provisioner = "csi.kubevirt.io"
)

func getDesiredStorageClass(obj metav1.Object, storageClass csiprovisionerv1alpha1.StorageClass) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kubevirt-%s", storageClass.InfraStorageClassName),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
		},
		Provisioner: provisioner,
		Parameters:  map[string]string{"infraStorageClassName": storageClass.InfraStorageClassName, "bus": storageClass.Bus},
	}
}

func (r *TenantReconciler) reconcileStorageClasses(ctx context.Context, obj metav1.Object, storageClasses []csiprovisionerv1alpha1.StorageClass) error {
	l := log.FromContext(ctx).WithName("storageClass")
	l.Info("Reconciling storageClass")
	for _, storageClass := range storageClasses {
		desiredStorageClass := getDesiredStorageClass(obj, storageClass)
		currentStorageClass := desiredStorageClass.DeepCopyObject().(*storagev1.StorageClass)
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, currentStorageClass, func() error {
			currentStorageClass.OwnerReferences = desiredStorageClass.OwnerReferences
			currentStorageClass.Parameters = desiredStorageClass.Parameters
			return nil

		}); err != nil {
			return err
		}
	}
	return nil
}
