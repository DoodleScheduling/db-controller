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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	infrav1beta1 "github.com/doodlescheduling/db-controller/api/v1beta1"
	// +kubebuilder:scaffold:imports
)

type mongodbContainer struct {
	testcontainers.Container
	URI string
}

func setupMongoDBContainer(ctx context.Context, image string) (*mongodbContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017"),
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "root",
			"MONGO_INITDB_ROOT_PASSWORD": "password",
		},
		Tmpfs: map[string]string{
			"/data/db": "",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, err
	}

	ip, err := container.ContainerIP(ctx)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("mongodb://%s:27017", ip)

	return &mongodbContainer{Container: container, URI: uri}, nil
}

var _ = Describe("MongoDB", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	for _, image := range []string{"mongo:5", "mongo:6"} {
		var _ = Describe(image, func() {
			var (
				container *mongodbContainer
				err       error
			)

			container, err = setupMongoDBContainer(context.TODO(), image)
			Expect(err).NotTo(HaveOccurred(), "failed to start mongodb container")

			Describe("fails if database not found", Ordered, func() {
				var (
					createdUser *infrav1beta1.MongoDBUser
					keyUser     types.NamespacedName
				)

				namespace, _ := setupNamespace()

				It("creates user", func() {
					keyUser = types.NamespacedName{
						Name:      "mongodbuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.MongoDBUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
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

					Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())
				})

				It("fails reconcile because there is no database", func() {
					got := &infrav1beta1.MongoDBUser{}
					Eventually(func() bool {
						_ = k8sClient.Get(context.Background(), keyUser, got)

						return len(got.Status.Conditions) == 1 &&
							got.Status.Conditions[0].Reason == infrav1beta1.DatabaseNotFoundReason &&
							got.Status.Conditions[0].Status == "False" &&
							got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType
					}, timeout, interval).Should(BeTrue())
				})
			})

			Describe("fails if datatabase root secret not found", Ordered, func() {
				var (
					createdDB   *infrav1beta1.MongoDBDatabase
					createdUser *infrav1beta1.MongoDBUser
					keyUser     types.NamespacedName
					keyDB       types.NamespacedName
				)

				namespace, _ := setupNamespace()

				It("creates database", func() {
					keyDB = types.NamespacedName{
						Name:      "mongodbdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.MongoDBDatabase{
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
				})

				It("adds user", func() {
					keyUser = types.NamespacedName{
						Name:      "mongodbuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.MongoDBUser{
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
				})

				It("fails reconcile because the db root secret is not found", func() {
					got := &infrav1beta1.MongoDBUser{}
					Eventually(func() bool {
						_ = k8sClient.Get(context.Background(), keyUser, got)

						return len(got.Status.Conditions) == 1 &&
							got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
							got.Status.Conditions[0].Status == "False" &&
							got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

					}, timeout, interval).Should(BeTrue())
				})
			})

			Describe("fails if user secret is not found", Ordered, func() {
				var (
					createdDB   *infrav1beta1.MongoDBDatabase
					createdUser *infrav1beta1.MongoDBUser
					keyUser     types.NamespacedName
					keyDB       types.NamespacedName
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "mongodbdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.MongoDBDatabase{
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
				})

				It("adds user", func() {
					keyUser = types.NamespacedName{
						Name:      "mongodbuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.MongoDBUser{
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
				})

				It("expects reconcile to fail because the user secret is not found", func() {
					got := &infrav1beta1.MongoDBUser{}
					Eventually(func() bool {
						_ = k8sClient.Get(context.Background(), keyUser, got)
						return len(got.Status.Conditions) == 1 &&
							got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
							got.Status.Conditions[0].Status == "False" &&
							got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

					}, timeout, interval).Should(BeTrue())
				})
			})

			Describe("fails if database can't be reached", Ordered, func() {
				var (
					createdDB     *infrav1beta1.MongoDBDatabase
					createdUser   *infrav1beta1.MongoDBUser
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					keySecret     types.NamespacedName
					password      string
					createdSecret *corev1.Secret
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "mongodbdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.MongoDBDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.MongoDBDatabaseSpec{
							DatabaseSpec: &infrav1beta1.DatabaseSpec{
								Timeout: &metav1.Duration{
									Duration: time.Millisecond * 100,
								},
								Address: container.URI,
								RootSecret: &infrav1beta1.SecretReference{
									Name: rootSecret.Name,
								},
							},
						},
					}
					Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())
				})

				It("adds secret", func() {
					keySecret = types.NamespacedName{
						Name:      "secret-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					password = randStringRunes(5)
					createdSecret = &corev1.Secret{
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
				})

				It("adds user", func() {
					keyUser = types.NamespacedName{
						Name:      "mongodbuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.MongoDBUser{
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
						},
					}
					Expect(k8sClient.Create(context.Background(), createdUser)).Should(Succeed())
				})

				It("fails reconcile because user credentials are not found", func() {
					got := &infrav1beta1.MongoDBUser{}
					Eventually(func() bool {
						_ = k8sClient.Get(context.Background(), keyUser, got)
						return len(got.Status.Conditions) == 1 &&
							got.Status.Conditions[0].Reason == infrav1beta1.ConnectionFailedReason &&
							got.Status.Conditions[0].Status == "False" &&
							got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType
					}, timeout, interval).Should(BeTrue())
				})
			})

			Describe("fails if user secret exists but fields not found", Ordered, func() {
				var (
					createdDB     *infrav1beta1.MongoDBDatabase
					createdUser   *infrav1beta1.MongoDBUser
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					createdSecret *v1.Secret
					keySecret     types.NamespacedName
					password      string
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "mongodbdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.MongoDBDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.MongoDBDatabaseSpec{
							DatabaseSpec: &infrav1beta1.DatabaseSpec{
								Timeout: &metav1.Duration{
									Duration: time.Millisecond * 100,
								},
								Address: container.URI,
								RootSecret: &infrav1beta1.SecretReference{
									Name: rootSecret.Name,
								},
							},
						},
					}
					Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())
				})

				It("adds secret", func() {
					keyUser = types.NamespacedName{
						Name:      "mongodbuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					keySecret = types.NamespacedName{
						Name:      "secret-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					password = randStringRunes(5)
					createdSecret = &v1.Secret{
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
				})

				It("adds user", func() {
					createdUser = &infrav1beta1.MongoDBUser{
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
				})

				It("fails reconcile because user field in secret is not found", func() {
					got := &infrav1beta1.MongoDBUser{}
					Eventually(func() bool {
						_ = k8sClient.Get(context.Background(), keyUser, got)
						return len(got.Status.Conditions) == 1 &&
							got.Status.Conditions[0].Reason == infrav1beta1.CredentialsNotFoundReason &&
							got.Status.Conditions[0].Status == "False" &&
							strings.Contains(got.Status.Conditions[0].Message, "credentials field not found in referenced secret:") &&
							got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

					}, timeout, interval).Should(BeTrue())
				})
			})

			Describe("Successful user creation", Ordered, func() {
				var (
					createdDB     *infrav1beta1.MongoDBDatabase
					createdUser   *infrav1beta1.MongoDBUser
					createdSecret *v1.Secret
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					keySecret     types.NamespacedName
					password      string
				)

				namespace, rootSecret := setupNamespace()

				Describe("creates readWrite user if it does not exists", Ordered, func() {
					var (
						client *mongo.Client
					)

					It("adds database", func() {
						keyDB = types.NamespacedName{
							Name:      "mongodbdatabase-" + randStringRunes(5),
							Namespace: namespace.Name,
						}
						createdDB = &infrav1beta1.MongoDBDatabase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      keyDB.Name,
								Namespace: keyDB.Namespace,
							},
							Spec: infrav1beta1.MongoDBDatabaseSpec{
								DatabaseSpec: &infrav1beta1.DatabaseSpec{
									Timeout: &metav1.Duration{
										Duration: time.Millisecond * 100,
									},
									Address: container.URI,
									RootSecret: &infrav1beta1.SecretReference{
										Name: rootSecret.Name,
									},
								},
							},
						}

						Expect(k8sClient.Create(context.Background(), createdDB)).Should(Succeed())
					})

					It("adds user", func() {
						keySecret = types.NamespacedName{
							Name:      "secret-" + randStringRunes(5),
							Namespace: namespace.Name,
						}
						keyUser = types.NamespacedName{
							Name:      "mongodbuser-" + randStringRunes(5),
							Namespace: namespace.Name,
						}
						createdUser = &infrav1beta1.MongoDBUser{
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
					})

					It("adds secret", func() {
						password = randStringRunes(5)
						createdSecret = &v1.Secret{
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
					})

					It("expects ready user", func() {
						got := &infrav1beta1.MongoDBUser{}
						Eventually(func() bool {
							_ = k8sClient.Get(context.Background(), keyUser, got)
							return len(got.Status.Conditions) == 1 &&
								got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
								got.Status.Conditions[0].Status == "True" &&
								got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

						}, timeout, interval).Should(BeTrue())
					})

					It("can access the created database", func() {
						o := options.Client()
						o.SetConnectTimeout(time.Duration(1) * time.Second)
						o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
						o.ApplyURI(container.URI)
						o.SetAuth(options.Credential{
							AuthSource: createdDB.ObjectMeta.Name,
							Username:   keyUser.Name,
							Password:   password,
						})

						client, err = mongo.Connect(ctx, o)
						Expect(err).NotTo(HaveOccurred(), "failed to connecto to mongodb")

						Eventually(func() error {
							return client.Ping(ctx, readpref.Primary())
						}, timeout, interval).Should(Succeed())
					})

					It("has write access to the referenced role database", func() {
						_, err = client.Database("foo").Collection(randStringRunes(5)).InsertOne(ctx, bson.D{})
						Expect(err).NotTo(HaveOccurred(), "failed to insert doc")
					})

					It("has has no access to another database", func() {
						_, err = client.Database("bar").Collection(randStringRunes(5)).InsertOne(ctx, bson.D{})
						Expect(err).To(HaveOccurred(), "failed to insert doc")
					})

					It("can't access the created database with invalid credentials", func() {
						o := options.Client()
						o.SetConnectTimeout(time.Duration(1) * time.Second)
						o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
						o.ApplyURI(container.URI)
						o.SetAuth(options.Credential{
							AuthSource: createdDB.ObjectMeta.Name,
							Username:   keyUser.Name,
							Password:   "invalid",
						})

						client, err = mongo.Connect(ctx, o)
						Expect(err).NotTo(HaveOccurred(), "failed to connecto to mongodb")

						Eventually(func() error {
							return client.Ping(ctx, readpref.Primary())
						}, timeout, interval).ShouldNot(Succeed())
					})
				})

				Describe("Change password for user", Ordered, func() {
					It("changes password in referenced user secret", func() {
						password = randStringRunes(5)
						createdSecret.Data = map[string][]byte{
							"username": []byte(createdUser.ObjectMeta.Name),
							"password": []byte(password),
						}
						Expect(k8sClient.Update(context.Background(), createdSecret)).Should(Succeed())
					})

					It("expects ready user", func() {
						got := &infrav1beta1.MongoDBUser{}
						Eventually(func() bool {
							_ = k8sClient.Get(context.Background(), keyUser, got)
							return len(got.Status.Conditions) == 1 &&
								got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
								got.Status.Conditions[0].Status == "True" &&
								got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

						}, timeout, interval).Should(BeTrue())
					})

					It("can access the database with the new password", func() {
						o := options.Client()
						o.SetConnectTimeout(time.Duration(1) * time.Second)
						o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
						o.ApplyURI(container.URI)

						o.SetAuth(options.Credential{
							AuthSource: createdDB.ObjectMeta.Name,
							Username:   createdUser.ObjectMeta.Name,
							Password:   password,
						})

						client, err := mongo.Connect(ctx, o)
						Expect(err).NotTo(HaveOccurred(), "failed to connecto to mongodb")

						Eventually(func() error {
							return client.Ping(ctx, readpref.Primary())
						}, timeout, interval).Should(Succeed())
					})
				})

				Describe("Change role", Ordered, func() {
					var (
						client *mongo.Client
					)

					It("changes role to readOnly", func() {
						err := k8sClient.Get(context.Background(), keyUser, createdUser)
						Expect(err).Should(Succeed())

						createdUser.Spec.Roles = &[]infrav1beta1.MongoDBUserRole{
							infrav1beta1.MongoDBUserRole{
								Name: "read",
								DB:   "foo",
							},
						}

						Expect(k8sClient.Update(context.Background(), createdUser)).Should(Succeed())
					})

					It("expects ready user", func() {
						got := &infrav1beta1.MongoDBUser{}
						Eventually(func() bool {
							_ = k8sClient.Get(context.Background(), keyUser, got)

							return len(got.Status.Conditions) == 1 &&
								got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
								got.Status.Conditions[0].Status == "True" &&
								got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType &&
								got.ObjectMeta.Generation == got.Status.ObservedGeneration

						}, timeout, interval).Should(BeTrue())
					})

					It("can't insert doc", func() {
						o := options.Client()
						o.SetConnectTimeout(time.Duration(1) * time.Second)
						o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
						o.ApplyURI(container.URI)

						o.SetAuth(options.Credential{
							AuthSource: createdDB.ObjectMeta.Name,
							Username:   createdUser.ObjectMeta.Name,
							Password:   password,
						})

						client, err = mongo.Connect(ctx, o)
						Expect(err).NotTo(HaveOccurred(), "failed to connecto to mongodb")

						Eventually(func() error {
							_, err = client.Database("foo").Collection(randStringRunes(5)).InsertOne(ctx, bson.D{})
							return err
						}, timeout, interval).ShouldNot(Succeed())
					})

					It("can still read though", func() {
						Eventually(func() error {
							_, err := client.Database("foo").Collection(randStringRunes(5)).Find(ctx, bson.D{})
							return err
						}, timeout, interval).Should(Succeed())
					})
				})

				Describe("Delete user removes user from mongodb", Ordered, func() {
					var (
						client *mongo.Client
					)

					It("deletes user", func() {
						Expect(k8sClient.Delete(context.Background(), createdUser)).Should(Succeed())
					})

					It("expects gone", func() {
						got := &infrav1beta1.MongoDBUser{}
						Eventually(func() error {
							return k8sClient.Get(context.Background(), keyUser, got)
						}, timeout, interval).ShouldNot(Succeed())
					})

					It("can't authenticate anymore since the user is deleted", func() {
						o := options.Client()
						o.SetConnectTimeout(time.Duration(1) * time.Second)
						o.SetServerSelectionTimeout(time.Duration(1) * time.Second)
						o.ApplyURI(container.URI)

						o.SetAuth(options.Credential{
							AuthSource: createdDB.ObjectMeta.Name,
							Username:   createdUser.ObjectMeta.Name,
							Password:   password,
						})

						client, err = mongo.Connect(ctx, o)
						Expect(err).NotTo(HaveOccurred(), "failed to connect to mongodb")

						Eventually(func() error {
							return client.Ping(ctx, readpref.Primary())
						}, timeout, interval).ShouldNot(Succeed())
					})
				})
			})
		})
	}
})
