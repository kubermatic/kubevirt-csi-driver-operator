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
	"fmt"
	"strconv"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	provisioner                        = "csi.kubevirt.io"
	isDefaultStorageClassannotationKey = "storageclass.kubernetes.io/is-default-class"
)

func getDesiredStorageClass(obj metav1.Object, storageClass csiprovisionerv1alpha1.StorageClass) *storagev1.StorageClass {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kubevirt-%s", storageClass.InfraStorageClassName),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
			Annotations: map[string]string{
				isDefaultStorageClassannotationKey: strconv.FormatBool(storageClass.IsDefaultClass != nil && *storageClass.IsDefaultClass),
			},
			Labels: storageClass.Labels,
		},
		Provisioner: provisioner,
		Parameters: map[string]string{
			"infraStorageClassName": storageClass.InfraStorageClassName,
			"bus":                   storageClass.Bus},
		VolumeBindingMode: storageClass.VolumeBindingMode,
	}

	var allowedTopologies []corev1.TopologySelectorTerm
	if len(storageClass.Zones) > 0 {
		allowedTopology := corev1.TopologySelectorTerm{
			MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
				{
					Key:    "topology.kubernetes.io/zone",
					Values: storageClass.Zones,
				},
			},
		}
		allowedTopologies = append(allowedTopologies, allowedTopology)
	}

	if len(storageClass.Regions) > 0 {
		allowedTopology := corev1.TopologySelectorTerm{
			MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
				{
					Key:    "topology.kubernetes.io/region",
					Values: storageClass.Regions,
				},
			},
		}
		allowedTopologies = append(allowedTopologies, allowedTopology)
	}

	sc.AllowedTopologies = allowedTopologies

	return sc
}

func (r *TenantReconciler) reconcileStorageClasses(ctx context.Context, obj metav1.Object, storageClasses []csiprovisionerv1alpha1.StorageClass) error {
	l := log.FromContext(ctx).WithName("storageClass")
	l.Info("Reconciling storageClass")
	for _, storageClass := range storageClasses {
		desiredStorageClass := getDesiredStorageClass(obj, storageClass)
		currentStorageClass := desiredStorageClass.DeepCopyObject().(*storagev1.StorageClass)
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, currentStorageClass, func() error {
			currentStorageClass.Annotations = desiredStorageClass.Annotations
			currentStorageClass.OwnerReferences = desiredStorageClass.OwnerReferences
			currentStorageClass.Parameters = desiredStorageClass.Parameters
			currentStorageClass.VolumeBindingMode = desiredStorageClass.VolumeBindingMode
			return nil

		}); err != nil {
			return err
		}
	}
	return nil
}
