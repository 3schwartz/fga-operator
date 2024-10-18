package interfaces

import (
	v12 "fga-operator/api/v1"
	v1 "k8s.io/api/apps/v1"
)

type AuthorizationModelInterface interface {
	GetVersionFromDeployment(deployment v1.Deployment) (v12.AuthorizationModelInstance, error)
}
