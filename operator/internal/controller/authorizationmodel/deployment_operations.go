package authorizationmodel

import (
	extensionsv1 "fga-operator/api/v1"
	"fga-operator/internal/interfaces"
	"github.com/go-logr/logr"
	appsV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type DeploymentIdentifier struct {
	namespace string
	name      string
}

func updateStoreIdOnDeployments(
	deployments appsV1.DeploymentList,
	store *extensionsv1.Store,
	reconcileTimestamp time.Time,
) map[DeploymentIdentifier]appsV1.Deployment {
	updates := map[DeploymentIdentifier]appsV1.Deployment{}
	for _, deployment := range deployments.Items {
		if updateDeploymentEnvVar(&deployment, extensionsv1.OpenFgaStoreIdEnv, store.Spec.Id) {
			if deployment.Annotations == nil {
				deployment.Annotations = make(map[string]string)
			}
			deployment.Annotations[extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation] = reconcileTimestamp.UTC().Format(time.RFC3339)

			updates[DeploymentIdentifier{namespace: deployment.Namespace, name: deployment.Name}] = deployment
		}
	}

	return updates
}

func updateAuthorizationModelIdOnDeployment(
	deployments appsV1.DeploymentList,
	updates map[DeploymentIdentifier]appsV1.Deployment,
	authorizationModel interfaces.AuthorizationModelInterface,
	reconcileTimestamp time.Time,
	log *logr.Logger,
) {
	for _, deployment := range deployments.Items {
		authInstance, err := authorizationModel.GetVersionFromDeployment(deployment)
		if err != nil {
			// TODO: make event
			log.Error(err, "unable to get auth instance from deployment", "deploymentName", deployment.Name)
			continue
		}
		deploymentIdentifier := DeploymentIdentifier{namespace: deployment.Namespace, name: deployment.Name}
		if updatedDeployment, ok := updates[deploymentIdentifier]; ok {
			deployment = updatedDeployment
		}
		if !updateDeploymentEnvVar(&deployment, extensionsv1.OpenFgaAuthModelIdEnv, authInstance.Id) {
			log.V(1).Info("deployment had correct auth id", "authInstance", authInstance)
			continue
		}

		deployment.Annotations[extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation] = reconcileTimestamp.UTC().Format(time.RFC3339)
		deployment.Annotations[extensionsv1.OpenFgaAuthModelVersionLabel] = authInstance.Version.String()

		updates[deploymentIdentifier] = deployment
	}
}

func updateDeploymentEnvVar(deployment *appsV1.Deployment, envVarName, envVarValue string) bool {
	updated := false
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		hasEnv := false
		for j := range container.Env {
			env := &container.Env[j]
			if env.Name != envVarName {
				continue
			}
			hasEnv = true
			if env.Value != envVarValue {
				updated = true
				env.Value = envVarValue
			}
			break
		}
		if hasEnv {
			continue
		}
		container.Env = append(container.Env, corev1.EnvVar{Name: envVarName, Value: envVarValue})
		updated = true
	}

	return updated
}
