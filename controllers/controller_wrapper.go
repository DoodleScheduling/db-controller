package controllers

import (
	"context"
	"errors"
	"strings"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// errors
var (
	ErrNoRootSecret = errors.New("there is no root secret field entry under specified rootSecret field")
)

type ControllerWrapper struct {
	wrapped *client.Reader
	ctx     *context.Context
}

func NewControllerWrapper(controller client.Reader, ctx *context.Context) *ControllerWrapper {
	return &ControllerWrapper{
		wrapped: &controller,
		ctx:     ctx,
	}
}

func (cw *ControllerWrapper) GetRootPassword(name string, namespace string, field string) (string, error) {
	var rootSecret v1.Secret
	if err := (*cw.wrapped).Get(*cw.ctx, types.NamespacedName{Name: name, Namespace: namespace}, &rootSecret); err != nil {
		return "", err
	}
	if len(rootSecret.Data[field]) == 0 {
		return "", ErrNoRootSecret
	}
	return string(rootSecret.Data[field][:]), nil
}

// objectKey returns client.ObjectKey for the object.
func objectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

func buildURI(uri string, secret *v1.Secret) string {
	for k, v := range secret.Data {
		uri = strings.Replace(uri, ("$" + k), string(v), 1)
	}

	return uri
}

func extractCredentials(credentials *infrav1beta1.SecretReference, secret *corev1.Secret) (string, string, error) {
	var (
		user string
		pw   string
	)

	if val, ok := secret.Data[credentials.UserField]; !ok {
		return "", "", errors.New("Defined username field not found in secret")
	} else {
		user = string(val)
	}

	if val, ok := secret.Data[credentials.PasswordField]; !ok {
		return "", "", errors.New("Defined password field not found in secret")
	} else {
		pw = string(val)
	}

	return user, pw, nil
}
