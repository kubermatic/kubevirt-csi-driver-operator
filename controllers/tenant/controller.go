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

	appsv1 "k8s.io/api/apps/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
)

const (
	namespaceName     = "kubevirt-csi-driver"
	tenantName        = "tenant"
	csiDeploymentName = "kubevirt-csi-controller"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=csiprovisioner.kubevirt.io,resources=tenants,verbs=get;list;watch
//+kubebuilder:rbac:groups=csiprovisioner.kubevirt.io,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csiprovisioner.kubevirt.io,resources=tenants/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=delete

//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs="*"
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims;persistentvolumeclaims/status,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups="",resources=serviceaccounts;events;configmaps,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups=extensions;apps,resources=daemonsets,verbs=get;list;watch;update;patch;create
//+kubebuilder:rbac:groups=storage.k8s.io,resources=volumeattachments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=volumeattachments/status,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses;,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=storage.k8s.io;csi.storage.k8s.io,resources=csinodes;csinodeinfos,verbs=get;list;watch
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs="*"
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Tenant", "name", req.Name)

	tenant := csiprovisionerv1alpha1.Tenant{}
	err := r.Client.Get(ctx, req.NamespacedName, &tenant)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("Tenant instance not found", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		l.Info("Error reading the request object, requeuing.")
		return ctrl.Result{}, err
	}
	objMeta := tenant.GetObjectMeta()

	_, err = r.reconcileCSIDriver(ctx, objMeta)
	if err != nil {
		l.Info("Error reconciling csi driver, requeuing.")
		return ctrl.Result{}, err
	}

	_, err = r.reconcileRBAC(ctx, objMeta)
	if err != nil {
		l.Info("Error reconciling rbac, requeuing.")
		return ctrl.Result{}, err
	}

	_, err = r.reconcileDaemonset(ctx, objMeta,
		tenant.Spec.ImageRepository, tenant.Spec.CSIDriverTag, tenant.Spec.CSISidecarTag)
	if err != nil {
		l.Info("Error reconciling daemonset, requeuing.")
		return ctrl.Result{}, err
	}

	err = r.reconcileStorageClasses(ctx, objMeta, tenant.Spec.StorageClasses)
	if err != nil {
		l.Info("Error reconciling storageClass, requeuing.")
		return ctrl.Result{}, err
	}

	// Cleanup the Deployment that is not removed during migration from non-split to split deployment
	err = r.Client.Delete(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      csiDeploymentName,
			Namespace: namespaceName,
		}})
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to ensure deployment %s is removed/not present: %w", csiDeploymentName, err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	filterTenants := predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return event.Object.GetName() == tenantName
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return event.ObjectNew.GetName() == tenantName
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return event.Object.GetName() == tenantName
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&csiprovisionerv1alpha1.Tenant{}, builder.WithPredicates(filterTenants)).
		Owns(&storagev1.CSIDriver{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&storagev1.StorageClass{}).
		Complete(r)
}
