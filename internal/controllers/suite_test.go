/*


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

package controllers

import (
	"context"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/doodlescheduling/db-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	cfg       *rest.Config
	k8sClient client.Client // You'll be using this client in your tests.
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "base", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	// MongoDBDatabase setup
	err = (&MongoDBDatabaseReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("MongoDBDatabase"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("MongoDBDatabase"),
	}).SetupWithManager(k8sManager, 1)

	Expect(err).ToNot(HaveOccurred(), "failed to setup MongoDBDatabase")

	// MongoDBUser setup
	err = (&MongoDBUserReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("MongoDBUser"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("MongoDBUser"),
	}).SetupWithManager(k8sManager, 1)
	Expect(err).ToNot(HaveOccurred(), "failed to setup MongoDBUser")

	// PostgreSQLDatabase setup
	err = (&PostgreSQLDatabaseReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("PostgreSQLDatabase"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("PostgreSQLDatabase"),
	}).SetupWithManager(k8sManager, 1)
	Expect(err).ToNot(HaveOccurred(), "failed to setup PostgreSQLDatabase")

	// PostgreSQLUser setup
	err = (&PostgreSQLUserReconciler{
		Client:   k8sManager.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("PostgreSQLUser"),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("PostgreSQLUser"),
	}).SetupWithManager(k8sManager, 1)
	Expect(err).ToNot(HaveOccurred(), "failed to setup PostgreSQLUser")

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func setupNamespace() (*v1.Namespace, *v1.Secret) {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "ns-" + randStringRunes(5)},
	}

	keyRootSecret := types.NamespacedName{
		Name:      "secret-" + randStringRunes(5),
		Namespace: namespace.Name,
	}
	createdSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keyRootSecret.Name,
			Namespace: keyRootSecret.Namespace,
		},
		Data: map[string][]byte{
			"username": []byte("root"),
			"password": []byte("password"),
		},
	}

	BeforeAll(func() {
		err := k8sClient.Create(context.Background(), namespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

		Expect(k8sClient.Create(context.Background(), createdSecret)).Should(Succeed())
	})

	AfterAll(func() {
		Eventually(func() error {
			return k8sClient.Delete(context.Background(), namespace)
		}, timeout, interval).Should(Succeed(), "failed to delete test namespace")
	})

	return namespace, createdSecret
}
