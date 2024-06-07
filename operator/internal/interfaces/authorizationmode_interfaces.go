package interfaces

import (
	v1 "k8s.io/api/apps/v1"
	v12 "openfga-controller/api/v1"
)

type AuthorizationModelInterface interface {
	GetVersionFromDeployment(deployment v1.Deployment) (v12.AuthorizationModelInstance, error)
}
