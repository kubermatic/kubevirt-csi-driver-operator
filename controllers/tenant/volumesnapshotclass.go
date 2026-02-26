/*
Copyright 2026 The KubeVirt CSI driver Operator Authors.

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
	"fmt"
	"strconv"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const isDefaultVolumeSnapshotClassAnnotationKey = "snapshot.storage.kubernetes.io/is-default-class"

func (r *TenantReconciler) reconcileVolumeSnapshotClasses(ctx context.Context, obj metav1.Object, volumeSnapshotClasses []csiprovisionerv1alpha1.VolumeSnapshotClass) error {
	l := log.FromContext(ctx).WithName("volumeSnapshotClass")
	l.Info("Reconciling volumeSnapshotClass")
	for _, volumeSnapshotClass := range volumeSnapshotClasses {
		deletionPolicy := volumeSnapshotClass.DeletionPolicy
		if volumeSnapshotClass.DeletionPolicy != "" {
			deletionPolicy = volumeSnapshotClass.DeletionPolicy
		}

		vsc := snapshotv1.VolumeSnapshotClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("kubevirt-%s", volumeSnapshotClass.InfraVolumeSnapshotClass),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
				},
				Annotations: map[string]string{
					isDefaultVolumeSnapshotClassAnnotationKey: strconv.FormatBool(volumeSnapshotClass.IsDefaultClass != nil && *volumeSnapshotClass.IsDefaultClass),
				},
			},
			Driver: "csi.kubevirt.io",
			Parameters: map[string]string{
				"infraSnapshotClassName": volumeSnapshotClass.InfraVolumeSnapshotClass,
			},
			DeletionPolicy: snapshotv1.DeletionPolicy(deletionPolicy),
		}

		currentVSC := vsc.DeepCopyObject().(*snapshotv1.VolumeSnapshotClass)
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, currentVSC, func() error {
			currentVSC.OwnerReferences = vsc.OwnerReferences
			currentVSC.DeletionPolicy = vsc.DeletionPolicy
			currentVSC.Annotations = vsc.Annotations
			currentVSC.Parameters = vsc.Parameters
			currentVSC.Driver = vsc.Driver
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}
