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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
)

func getDesiredDaemonsetClusterRole(obj metav1.Object) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: csiDaemonSetName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"list", "watch", "create", "update", "patch"},
			},
			{
				APIGroups: []string{"storage.k8s.io", "csi.storage.k8s.io"},
				Resources: []string{"csinodes", "csinodeinfos"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"volumeattachments", "volumeattachments/status"},
				Verbs:     []string{"get", "list", "watch", "update", "patch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func (r *TenantReconciler) reconcileRBAC(ctx context.Context, obj metav1.Object) (map[string]controllerutil.OperationResult, error) {
	l := log.FromContext(ctx).WithName("rbac")
	l.Info("Reconciling rbac")
	status := make(map[string]controllerutil.OperationResult)

	// daemonset
	desiredDaemonsetSa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      csiDaemonSetName,
			Namespace: namespaceName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
		},
	}
	currentDaemonsetSa := desiredDaemonsetSa.DeepCopyObject().(*corev1.ServiceAccount)
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, currentDaemonsetSa, func() error {
		currentDaemonsetSa.OwnerReferences = desiredDaemonsetSa.OwnerReferences
		return nil
	})
	if err != nil {
		return status, err
	}
	status["daemonset_service_account"] = op

	desiredDaemonsetCr := getDesiredDaemonsetClusterRole(obj)
	currentDaemonsetCr := desiredDaemonsetCr.DeepCopyObject().(*rbacv1.ClusterRole)
	op, err = ctrl.CreateOrUpdate(ctx, r.Client, currentDaemonsetCr, func() error {
		currentDaemonsetCr.OwnerReferences = desiredDaemonsetCr.OwnerReferences
		currentDaemonsetCr.Rules = desiredDaemonsetCr.Rules
		return nil
	})
	if err != nil {
		return status, err
	}
	status["daemonset_cluster_role"] = op

	desiredDaemonsetCrb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: csiDaemonSetName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(obj, csiprovisionerv1alpha1.GroupVersion.WithKind("Tenant")),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      csiDaemonSetName,
				Namespace: namespaceName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     csiDaemonSetName,
		},
	}
	currentDaemonsetCrb := desiredDaemonsetCrb.DeepCopyObject().(*rbacv1.ClusterRoleBinding)
	op, err = ctrl.CreateOrUpdate(ctx, r.Client, currentDaemonsetCrb, func() error {
		currentDaemonsetCrb.OwnerReferences = desiredDaemonsetCrb.OwnerReferences
		currentDaemonsetCrb.Subjects = desiredDaemonsetCrb.Subjects
		currentDaemonsetCrb.RoleRef = desiredDaemonsetCrb.RoleRef
		return nil
	})
	if err != nil {
		return status, err
	}
	status["daemonset_cluster_role_binding"] = op

	return status, nil
}
