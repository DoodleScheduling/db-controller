package controllers

import (
	"context"
	"errors"
	v12 "k8s.io/api/core/v1"
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
	var rootSecret v12.Secret
	if err := (*cw.wrapped).Get(*cw.ctx, types.NamespacedName{Name: name, Namespace: namespace}, &rootSecret); err != nil {
		return "", err
	}
	if len(rootSecret.Data[field]) == 0 {
		return "", ErrNoRootSecret
	}
	return string(rootSecret.Data[field][:]), nil
}
