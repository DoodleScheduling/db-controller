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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	// +kubebuilder:scaffold:imports
)

type mongodbContainer struct {
	testcontainers.Container
	URI string
}

func setupMongoDBContainer(ctx context.Context) (*mongodbContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:4.4",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017"),
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "root",
			"MONGO_INITDB_ROOT_PASSWORD": "password",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, err
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "27017")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("mongodb://%s:%s", ip, mappedPort.Port())

	return &mongodbContainer{Container: container, URI: uri}, nil
}

func setupMongoDBRootSecret(ctx context.Context, namespace *corev1.Namespace) *corev1.Secret {
	keyRootSecret := types.NamespacedName{
		Name:      "secret-" + randStringRunes(5),
		Namespace: namespace.Name,
	}
	createdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keyRootSecret.Name,
			Namespace: keyRootSecret.Namespace,
		},
		Data: map[string][]byte{
			"username": []byte("root"),
			"password": []byte("password"),
		},
	}

	Expect(k8sClient.Create(context.Background(), createdSecret)).Should(Succeed())
	return createdSecret
}

var _ = Describe("MongoDBUserReconciler", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	Context("MongoDBUser", func() {
		var (
			namespace  *corev1.Namespace
			container  *mongodbContainer
			rootSecret *corev1.Secret
			err        error
		)

		container, err = setupMongoDBContainer(context.TODO())
		Expect(err).NotTo(HaveOccurred(), "failed to start mongodb container")

		BeforeEach(func() {
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "mongodbuser-" + randStringRunes(5)},
			}
			err = k8sClient.Create(context.Background(), namespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

			rootSecret = setupMongoDBRootSecret(context.TODO(), namespace)
		})

		AfterEach(func() {
			Eventually(func() error {
				return k8sClient.Delete(context.Background(), namespace)
			}, timeout, interval).Should(Succeed(), "failed to delete test namespace")
		})

		It("fails if database not found", func() {
			key := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			created := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: "does-not-exist",
					},
					Credentials: &infrav1beta1.SecretReference{
						Name: "does-not-exist",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), created)).Should(Succeed())

			By("Expecting secret not found")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), key, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.DatabaseNotFoundReason &&
					got.Status.Conditions[0].Status == "False" &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType
			}, timeout, interval).Should(BeTrue())
		})

		It("fails if datatabse root secret not found", func() {
			By("adding database")
			keyDB := types.NamespacedName{
				Name:      "mongodbdatabase-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdDB := &infrav1beta1.MongoDBDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyDB.Name,
					Namespace: keyDB.Namespace,
				},
				Spec: infrav1beta1.MongoDBDatabaseSpec{
					DatabaseSpec: &infrav1beta1.DatabaseSpec{
						RootSecret: &infrav1beta1.SecretReference{
							Name: "does-not-exist",
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())

			keyUser := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdUser := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyUser.Name,
					Namespace: keyUser.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: keyDB.Name,
					},
					Credentials: &infrav1beta1.SecretReference{
						Name: "does-not-exist",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())

			By("Expecting secret not found")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), keyUser, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
					got.Status.Conditions[0].Status == "False" &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

			}, timeout, interval).Should(BeTrue())
		})

		It("fails if database is not ready", func() {
			By("adding database")
			keyDB := types.NamespacedName{
				Name:      "mongodbdatabase-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdDB := &infrav1beta1.MongoDBDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyDB.Name,
					Namespace: keyDB.Namespace,
				},
				Spec: infrav1beta1.MongoDBDatabaseSpec{
					DatabaseSpec: &infrav1beta1.DatabaseSpec{
						Address: "mongodb://does-not-exist:27017",
						RootSecret: &infrav1beta1.SecretReference{
							Name: rootSecret.Name,
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())

			By("Adding mogodbuser")
			keyUser := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdUser := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyUser.Name,
					Namespace: keyUser.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: keyDB.Name,
					},
					Credentials: &infrav1beta1.SecretReference{
						Name: "does-not-exist",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())

			By("Expecting database not ready")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), keyUser, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.ConnectionFailedReason &&
					got.Status.Conditions[0].Status == "False" &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

			}, timeout, interval).Should(BeTrue())
		})

		It("fails if user secret does not exist", func() {
			By("adding database")
			keyDB := types.NamespacedName{
				Name:      "mongodbdatabase-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdDB := &infrav1beta1.MongoDBDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyDB.Name,
					Namespace: keyDB.Namespace,
				},
				Spec: infrav1beta1.MongoDBDatabaseSpec{
					DatabaseSpec: &infrav1beta1.DatabaseSpec{
						Address: container.URI,
						RootSecret: &infrav1beta1.SecretReference{
							Name: rootSecret.Name,
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())

			By("Adding mogodbuser")
			keyUser := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdUser := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyUser.Name,
					Namespace: keyUser.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: keyDB.Name,
					},
					Credentials: &infrav1beta1.SecretReference{
						Name: "does-not-exist",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())

			By("Expecting user credentials not found")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), keyUser, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
					got.Status.Conditions[0].Status == "False" &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

			}, timeout, interval).Should(BeTrue())
		})

		It("fails if user secret exists but fields not found", func() {
			By("adding database")
			keyDB := types.NamespacedName{
				Name:      "mongodbdatabase-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdDB := &infrav1beta1.MongoDBDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyDB.Name,
					Namespace: keyDB.Namespace,
				},
				Spec: infrav1beta1.MongoDBDatabaseSpec{
					DatabaseSpec: &infrav1beta1.DatabaseSpec{
						Address: container.URI,
						RootSecret: &infrav1beta1.SecretReference{
							Name: rootSecret.Name,
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())

			By("Adding mogodbuser")
			keySecret := types.NamespacedName{
				Name:      "secret-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			keyUser := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdUser := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyUser.Name,
					Namespace: keyUser.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: keyDB.Name,
					},
					Credentials: &infrav1beta1.SecretReference{
						Name:      keySecret.Name,
						UserField: "does-not-exist",
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())

			By("Expecting user credentials not found")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), keyUser, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
					got.Status.Conditions[0].Status == "False" &&
					strings.Contains(got.Status.Conditions[0].Message, "Referencing secret was not found:") &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

			}, timeout, interval).Should(BeTrue())
		})

		It("creates r/w user if it does not exists", func() {
			By("adding database")
			keyDB := types.NamespacedName{
				Name:      "mongodbdatabase-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdDB := &infrav1beta1.MongoDBDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyDB.Name,
					Namespace: keyDB.Namespace,
				},
				Spec: infrav1beta1.MongoDBDatabaseSpec{
					DatabaseSpec: &infrav1beta1.DatabaseSpec{
						Address: container.URI,
						RootSecret: &infrav1beta1.SecretReference{
							Name: rootSecret.Name,
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())

			By("Adding mogodbuser")
			keySecret := types.NamespacedName{
				Name:      "secret-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			keyUser := types.NamespacedName{
				Name:      "mongodbuser-" + randStringRunes(5),
				Namespace: namespace.Name,
			}
			createdUser := &infrav1beta1.MongoDBUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keyUser.Name,
					Namespace: keyUser.Namespace,
				},
				Spec: infrav1beta1.MongoDBUserSpec{
					Database: &infrav1beta1.DatabaseReference{
						Name: keyDB.Name,
					},
					Credentials: &infrav1beta1.SecretReference{
						Name: keySecret.Name,
					},
					Roles: &[]infrav1beta1.MongoDBUserRole{
						infrav1beta1.MongoDBUserRole{
							Name: "readWrite",
							DB:   "foo",
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())

			By("Adding secret")
			password := randStringRunes(5)
			createdSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keySecret.Name,
					Namespace: keySecret.Namespace,
				},
				Data: map[string][]byte{
					"username": []byte(keyUser.Name),
					"password": []byte(password),
				},
			}
			Expect(k8sClient.Create(context.Background(), createdSecret)).Should(Succeed())

			By("Expecting user ready")
			got := &infrav1beta1.MongoDBUser{}
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), keyUser, got)
				return len(got.Status.Conditions) == 1 &&
					got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
					got.Status.Conditions[0].Status == "True" &&
					got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

			}, timeout, interval).Should(BeTrue())

			By("verify mongodb access")
			o := options.Client()
			o.SetConnectTimeout(time.Duration(1) * time.Second)
			o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
			o.ApplyURI(container.URI)

			o.SetAuth(options.Credential{
				Username: keyUser.Name,
				Password: password,
			})

			_, err := mongo.Connect(ctx, o)
			Expect(err).NotTo(HaveOccurred(), "failed to connecto to mongodb")
		})
	})
})
