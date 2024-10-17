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
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	infrav1beta1 "github.com/doodlescheduling/db-controller/api/v1beta1"
	// +kubebuilder:scaffold:imports
)

type postgresqlContainer struct {
	testcontainers.Container
	URI string
}

func setupPostgreSQLContainer(ctx context.Context, image string) (*postgresqlContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432"),
		Env: map[string]string{
			"POSTGRES_USER":     "root",
			"POSTGRES_PASSWORD": "password",
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

	uri := fmt.Sprintf("postgresql://%s:5432", ip)

	return &postgresqlContainer{Container: container, URI: uri}, nil
}

var _ = Describe("PostgreSQL", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	for _, image := range []string{"postgres:12", "postgres:13", "postgres:14", "postgres:15"} {
		var _ = Describe(image, func() {
			var (
				container *postgresqlContainer
				err       error
			)

			container, err = setupPostgreSQLContainer(context.Background(), image)
			Expect(err).NotTo(HaveOccurred(), "failed to start postgres container")

			Describe("fails if database not found", Ordered, func() {
				var (
					createdUser *infrav1beta1.PostgreSQLUser
					keyUser     types.NamespacedName
				)

				namespace, _ := setupNamespace()

				It("creates user", func() {
					keyUser = types.NamespacedName{
						Name:      "postgresuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.PostgreSQLUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLUserSpec{
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
					got := &infrav1beta1.PostgreSQLUser{}
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
					createdDB   *infrav1beta1.PostgreSQLDatabase
					createdUser *infrav1beta1.PostgreSQLUser
					keyUser     types.NamespacedName
					keyDB       types.NamespacedName
				)

				namespace, _ := setupNamespace()

				It("creates database", func() {
					keyDB = types.NamespacedName{
						Name:      "postgresdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.PostgreSQLDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLDatabaseSpec{
							DatabaseSpec: &infrav1beta1.DatabaseSpec{
								Timeout: &metav1.Duration{
									Duration: time.Millisecond * 100,
								},
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
						Name:      "postgresuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.PostgreSQLUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLUserSpec{
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
					got := &infrav1beta1.PostgreSQLUser{}
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
					createdDB   *infrav1beta1.PostgreSQLDatabase
					createdUser *infrav1beta1.PostgreSQLUser
					keyUser     types.NamespacedName
					keyDB       types.NamespacedName
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "postgresdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.PostgreSQLDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLDatabaseSpec{
							DatabaseSpec: &infrav1beta1.DatabaseSpec{
								Timeout: &metav1.Duration{
									Duration: time.Millisecond * 100,
								},
								Address: "postgres://does-not-exist:5432",
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
						Name:      "postgresuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.PostgreSQLUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLUserSpec{
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
					got := &infrav1beta1.PostgreSQLUser{}
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
					createdDB     *infrav1beta1.PostgreSQLDatabase
					createdUser   *infrav1beta1.PostgreSQLUser
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					keySecret     types.NamespacedName
					password      string
					createdSecret *corev1.Secret
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "postgresdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.PostgreSQLDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLDatabaseSpec{
							DatabaseSpec: &infrav1beta1.DatabaseSpec{
								Timeout: &metav1.Duration{
									Duration: time.Millisecond * 100,
								},
								Address: "postgres://does-not-exist:5432",
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
						Name:      "postgresuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.PostgreSQLUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLUserSpec{
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

				It("fails reconcile because database can't be reached", func() {
					got := &infrav1beta1.PostgreSQLUser{}
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
					createdDB     *infrav1beta1.PostgreSQLDatabase
					createdUser   *infrav1beta1.PostgreSQLUser
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					createdSecret *corev1.Secret
					keySecret     types.NamespacedName
					password      string
				)

				namespace, rootSecret := setupNamespace()

				It("adds database", func() {
					keyDB = types.NamespacedName{
						Name:      "postgresdatabase-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdDB = &infrav1beta1.PostgreSQLDatabase{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyDB.Name,
							Namespace: keyDB.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLDatabaseSpec{
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
						Name:      "postgresuser-" + randStringRunes(5),
						Namespace: namespace.Name,
					}
					createdUser = &infrav1beta1.PostgreSQLUser{
						ObjectMeta: metav1.ObjectMeta{
							Name:      keyUser.Name,
							Namespace: keyUser.Namespace,
						},
						Spec: infrav1beta1.PostgreSQLUserSpec{
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

				It("adds secret", func() {
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

				It("fails reconcile because user field in secret is not found", func() {
					got := &infrav1beta1.PostgreSQLUser{}
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
					createdDB     *infrav1beta1.PostgreSQLDatabase
					createdUser   *infrav1beta1.PostgreSQLUser
					createdSecret *corev1.Secret
					keyUser       types.NamespacedName
					keyDB         types.NamespacedName
					keySecret     types.NamespacedName
					password      string
				)

				namespace, rootSecret := setupNamespace()

				Describe("creates readWrite user if it does not exists", Ordered, func() {
					It("adds database", func() {
						keyDB = types.NamespacedName{
							Name:      "postgresdatabase-" + randStringRunes(5),
							Namespace: namespace.Name,
						}
						createdDB = &infrav1beta1.PostgreSQLDatabase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      keyDB.Name,
								Namespace: keyDB.Namespace,
							},
							Spec: infrav1beta1.PostgreSQLDatabaseSpec{
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
							Name:      "postgresuser-" + randStringRunes(5),
							Namespace: namespace.Name,
						}
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
						createdUser = &infrav1beta1.PostgreSQLUser{
							ObjectMeta: metav1.ObjectMeta{
								Name:      keyUser.Name,
								Namespace: keyUser.Namespace,
							},
							Spec: infrav1beta1.PostgreSQLUserSpec{
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

					It("expects ready user", func() {
						got := &infrav1beta1.PostgreSQLUser{}
						Eventually(func() bool {
							_ = k8sClient.Get(context.Background(), keyUser, got)
							return len(got.Status.Conditions) == 1 &&
								got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
								got.Status.Conditions[0].Status == "True" &&
								got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

						}, timeout, interval).Should(BeTrue())
					})

					It("can access the created database", func() {
						popt, err := url.Parse(container.URI)
						Expect(err).NotTo(HaveOccurred(), "failed to parse postgresql uri")

						popt.User = url.UserPassword(keyUser.Name, password)
						q, _ := url.ParseQuery(popt.RawQuery)
						q.Add("connect_timeout", "2")
						popt.RawQuery = q.Encode()
						popt.Path = keyDB.Name

						Expect(err).NotTo(HaveOccurred(), "failed to connect to postgresql")

						var client *pgx.Conn

						Eventually(func() error {
							c, err := pgx.Connect(ctx, popt.String())
							client = c
							return err
						}, timeout, interval).Should(Succeed())

						_, err = client.Exec(ctx, fmt.Sprintln("CREATE TABLE foo (key integer);"))
						Expect(err).NotTo(HaveOccurred(), "failed to insert doc")
					})

					It("has has no access to another database", func() {
						popt, err := url.Parse(container.URI)
						Expect(err).NotTo(HaveOccurred(), "failed to parse postgresql uri")

						popt.User = url.UserPassword(keyUser.Name, password)
						q, _ := url.ParseQuery(popt.RawQuery)
						q.Add("connect_timeout", "2")
						popt.RawQuery = q.Encode()
						popt.Path = "does-not-exist"

						Expect(err).NotTo(HaveOccurred(), "failed to connect to postgresql")

						Eventually(func() error {
							_, err := pgx.Connect(ctx, popt.String())
							return err
						}, timeout, interval).ShouldNot(Succeed())
					})

					It("can't access the created database with invalid credentials", func() {
						popt, err := url.Parse(container.URI)
						Expect(err).NotTo(HaveOccurred(), "failed to parse postgresql uri")

						popt.User = url.UserPassword(keyUser.Name, "invalid-password")
						q, _ := url.ParseQuery(popt.RawQuery)
						q.Add("connect_timeout", "2")
						popt.RawQuery = q.Encode()
						popt.Path = keyDB.Name

						Expect(err).NotTo(HaveOccurred(), "failed to connect to postgresql")

						Eventually(func() error {
							_, err := pgx.Connect(ctx, popt.String())
							return err
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
						got := &infrav1beta1.PostgreSQLUser{}
						Eventually(func() bool {
							_ = k8sClient.Get(context.Background(), keyUser, got)
							return len(got.Status.Conditions) == 1 &&
								got.Status.Conditions[0].Reason == infrav1beta1.UserProvisioningSuccessfulReason &&
								got.Status.Conditions[0].Status == "True" &&
								got.Status.Conditions[0].Type == infrav1beta1.UserReadyConditionType

						}, timeout, interval).Should(BeTrue())
					})

					It("can access the database with the new password", func() {
						popt, err := url.Parse(container.URI)
						Expect(err).NotTo(HaveOccurred(), "failed to parse postgresql uri")

						popt.User = url.UserPassword(keyUser.Name, password)
						q, _ := url.ParseQuery(popt.RawQuery)
						q.Add("connect_timeout", "2")
						popt.RawQuery = q.Encode()
						popt.Path = keyDB.Name

						Expect(err).NotTo(HaveOccurred(), "failed to connect to postgresql")

						Eventually(func() error {
							_, err := pgx.Connect(ctx, popt.String())
							return err
						}, timeout, interval).Should(Succeed())
					})
				})

				Describe("Delete user removes user from postgres", Ordered, func() {
					It("deletes user", func() {
						Expect(k8sClient.Delete(context.Background(), createdUser)).Should(Succeed())
					})

					It("expects gone", func() {
						got := &infrav1beta1.PostgreSQLUser{}
						Eventually(func() error {
							return k8sClient.Get(context.Background(), keyUser, got)
						}, timeout, interval).ShouldNot(Succeed())
					})

					It("can't authenticate anymore since the user is deleted", func() {
						popt, err := url.Parse(container.URI)
						Expect(err).NotTo(HaveOccurred(), "failed to parse postgresql uri")

						popt.User = url.UserPassword(keyUser.Name, password)
						q, _ := url.ParseQuery(popt.RawQuery)
						q.Add("connect_timeout", "2")
						popt.RawQuery = q.Encode()
						popt.Path = keyDB.Name

						Expect(err).NotTo(HaveOccurred(), "failed to connect to postgresql")

						Eventually(func() error {
							_, err := pgx.Connect(ctx, popt.String())
							return err
						}, timeout, interval).ShouldNot(Succeed())
					})
				})
			})
		})
	}
})
