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
	"path/filepath"
	"testing"

	"github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"
	csiprovisionerv1alpha1 "github.com/kubermatic/kubevirt-csi-driver-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager

var testEnv *envtest.Environment
var testCtx context.Context
var testCancel context.CancelFunc

var (
	testTenantName      = tenantName
	otherTestTenantName = fmt.Sprintf("other-%s", tenantName)
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	// any needed
})

func newTestTenant(name string) *v1alpha1.Tenant {
	return &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func startTestEnv() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testCtx, testCancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = csiprovisionerv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = storagev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = rbacv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

}

func startTestEnvWithReconciler() {
	startTestEnv()
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&TenantReconciler{
		Client: k8sManager.GetClient(),
		Scheme: scheme.Scheme,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
}

func stopTestEnv() {
	testCancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
}

func createTestTenant(t *v1alpha1.Tenant) {
	GinkgoHelper()

	// Needed to have a UID in the Tenant object.
	Expect(k8sClient.Create(testCtx, t)).To(Succeed())
	Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: t.Name}, t)).To(Succeed())
}

func cleanupTenants() {
	GinkgoHelper()

	By("Clean Tenants")
	l := &v1alpha1.TenantList{}
	Expect(k8sClient.List(testCtx, l)).To(Succeed())
	for _, i := range l.Items {
		Expect(k8sClient.Delete(testCtx, &i)).To(Succeed())
	}
}

func deleteTestTenant(name string) {
	GinkgoHelper()

	t := &v1alpha1.Tenant{}
	k8sClient.Get(testCtx, types.NamespacedName{
		Name: name}, t)
	Expect(k8sClient.Delete(testCtx, t)).To(Succeed())
}
