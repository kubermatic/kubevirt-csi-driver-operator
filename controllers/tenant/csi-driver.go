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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
)

const csiDriverName = "csi.kubevirt.io"

func getDesiredCSIDriverObj(obj metav1.Object) *storagev1.CSIDriver {
	return &storagev1.CSIDriver{
		ObjectMeta: metav1.ObjectMeta{
			Name: csiDriverName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
		},
		Spec: storagev1.CSIDriverSpec{
			AttachRequired: pointer.Bool(true),
			PodInfoOnMount: pointer.Bool(true),
		},
	}
}

func (r *TenantReconciler) reconcileCSIDriver(ctx context.Context, obj metav1.Object) (controllerutil.OperationResult, error) {
	l := log.FromContext(ctx).WithName("csi-driver")
	l.Info("Reconciling csi driver", "name", csiDriverName)

	desiredCSIObj := getDesiredCSIDriverObj(obj)
	currentCSIObj := desiredCSIObj.DeepCopyObject().(*storagev1.CSIDriver)
	return ctrl.CreateOrUpdate(ctx, r.Client, currentCSIObj, func() error {
		currentCSIObj.OwnerReferences = desiredCSIObj.OwnerReferences
		return nil
	})
}
