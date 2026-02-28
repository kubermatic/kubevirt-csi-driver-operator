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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const isDefaultVolumeSnapshotClassAnnotationKey = "snapshot.storage.kubernetes.io/is-default-class"

func (r *TenantReconciler) reconcileVolumeSnapshotClasses(ctx context.Context, obj metav1.Object, volumeSnapshotClasses []csiprovisionerv1alpha1.VolumeSnapshotClass) error {
	l := log.FromContext(ctx).WithName("volumeSnapshotClass")
	l.Info("Reconciling volumeSnapshotClass")
	for _, volumeSnapshotClass := range volumeSnapshotClasses {
		deletionPolicy := snapshotv1.VolumeSnapshotContentDelete
		if volumeSnapshotClass.DeletionPolicy != "" {
			deletionPolicy = snapshotv1.DeletionPolicy(volumeSnapshotClass.DeletionPolicy)
		}

		desiredVSC := &snapshotv1.VolumeSnapshotClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("kubevirt-%s", volumeSnapshotClass.InfraVolumeSnapshotClass),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
				},
				Annotations: map[string]string{
					isDefaultVolumeSnapshotClassAnnotationKey: strconv.FormatBool(volumeSnapshotClass.IsDefaultClass != nil && *volumeSnapshotClass.IsDefaultClass),
				},
			},
			Driver: provisioner,
			Parameters: map[string]string{
				"infraSnapshotClassName": volumeSnapshotClass.InfraVolumeSnapshotClass,
			},
			DeletionPolicy: deletionPolicy,
		}

		existingVSC := &snapshotv1.VolumeSnapshotClass{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: desiredVSC.Name}, existingVSC)
		if apierrors.IsNotFound(err) {
			l.Info("Creating VolumeSnapshotClass", "name", desiredVSC.Name)
			if err := r.Client.Create(ctx, desiredVSC); err != nil {
				return fmt.Errorf("failed to create VolumeSnapshotClass %s: %w", desiredVSC.Name, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to get VolumeSnapshotClass %s: %w", desiredVSC.Name, err)
		} else {
			existingVSC.Annotations = desiredVSC.Annotations
			existingVSC.OwnerReferences = desiredVSC.OwnerReferences
			existingVSC.Parameters = desiredVSC.Parameters
			existingVSC.DeletionPolicy = desiredVSC.DeletionPolicy
			existingVSC.Driver = desiredVSC.Driver
			l.Info("Updating VolumeSnapshotClass", "name", desiredVSC.Name)
			if err := r.Client.Update(ctx, existingVSC); err != nil {
				return fmt.Errorf("failed to update VolumeSnapshotClass %s: %w", desiredVSC.Name, err)
			}
		}
	}

	return nil
}
