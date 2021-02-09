package common

import "github.com/doodlescheduling/kubedb/api/v1beta1"

type DatabaseCredentials []DatabaseCredential
type DatabaseCredential struct {
	UserName string        `json:"username"`
	Vault    v1beta1.Vault `json:"vault"`
}
