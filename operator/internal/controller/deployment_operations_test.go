package controller

import (
	"context"
	"errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"

	extensionsv1 "fga-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"github.com/google/go-cmp/cmp"
	appsV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type MockAuthorizationModel struct{}

func (m *MockAuthorizationModel) GetVersionFromDeployment(deployment appsV1.Deployment) (extensionsv1.AuthorizationModelInstance, error) {
	if deployment.Name == "error-deployment" {
		return extensionsv1.AuthorizationModelInstance{}, errors.New("mock error")
	}
	return extensionsv1.AuthorizationModelInstance{
		Id: "auth-model-id",
		Version: extensionsv1.ModelVersion{
			Major: 1,
			Minor: 2,
			Patch: 3,
		},
	}, nil
}

func createDeployment(envVars []corev1.EnvVar) *appsV1.Deployment {
	return &appsV1.Deployment{
		Spec: appsV1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							Env:  envVars,
						},
					},
				},
			},
		},
	}
}

func createDeploymentWithNameAndAnnotations(namespace, name string, envVars []corev1.EnvVar, annotations map[string]string) appsV1.Deployment {
	return appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Annotations: annotations,
		},
		Spec: appsV1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							Env:  envVars,
						},
					},
				},
			},
		},
	}
}

func TestUpdateStoreIdOnDeployments(t *testing.T) {
	store := &extensionsv1.Store{
		Spec: extensionsv1.StoreSpec{
			Id: "store-id",
		},
	}
	reconcileTimestamp := time.Now()
	reconcileTimestampFormatted := reconcileTimestamp.UTC().Format(time.RFC3339)

	tests := []struct {
		name               string
		deployments        appsV1.DeploymentList
		store              *extensionsv1.Store
		reconcileTimestamp time.Time
		expectedUpdates    map[DeploymentIdentifier]appsV1.Deployment
	}{
		{
			name: "No deployments",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{},
			},
			store:              store,
			reconcileTimestamp: reconcileTimestamp,
			expectedUpdates:    map[DeploymentIdentifier]appsV1.Deployment{},
		},
		{
			name: "Deployment without the target env var",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations("namespace1", "deployment1", []corev1.EnvVar{}, map[string]string{}),
				},
			},
			store:              store,
			reconcileTimestamp: reconcileTimestamp,
			expectedUpdates: map[DeploymentIdentifier]appsV1.Deployment{
				{namespace: "namespace1", name: "deployment1"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"deployment1",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaStoreIdEnv, Value: "store-id"},
					},
					map[string]string{
						extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation: reconcileTimestampFormatted,
					},
				),
			},
		},
		{
			name: "Deployment with the target env var but different value",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations(
						"namespace1",
						"deployment1",
						[]corev1.EnvVar{
							{Name: extensionsv1.OpenFgaStoreIdEnv, Value: "old-value"},
						},
						map[string]string{},
					),
				},
			},
			store:              store,
			reconcileTimestamp: reconcileTimestamp,
			expectedUpdates: map[DeploymentIdentifier]appsV1.Deployment{
				{namespace: "namespace1", name: "deployment1"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"deployment1",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaStoreIdEnv, Value: "store-id"},
					},
					map[string]string{
						extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation: reconcileTimestampFormatted,
					},
				),
			},
		},
		{
			name: "Deployment with the target env var already set to correct value",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations(
						"namespace1",
						"deployment1",
						[]corev1.EnvVar{
							{Name: extensionsv1.OpenFgaStoreIdEnv, Value: "store-id"},
						},
						map[string]string{},
					),
				},
			},
			store:              store,
			reconcileTimestamp: reconcileTimestamp,
			expectedUpdates:    map[DeploymentIdentifier]appsV1.Deployment{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := updateStoreIdOnDeployments(tt.deployments, tt.store, tt.reconcileTimestamp)

			if diff := cmp.Diff(tt.expectedUpdates, updates); diff != "" {
				t.Errorf("unexpected updates (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateDeploymentEnvVar(t *testing.T) {
	tests := []struct {
		name           string
		initialEnvVars []corev1.EnvVar
		envVarName     string
		envVarValue    string
		expectedEnv    []corev1.EnvVar
		expectedUpdate bool
	}{
		{
			name:           "EnvVar does not exist and should be added",
			initialEnvVars: []corev1.EnvVar{},
			envVarName:     "NEW_VAR",
			envVarValue:    "new_value",
			expectedEnv: []corev1.EnvVar{
				{Name: "NEW_VAR", Value: "new_value"},
			},
			expectedUpdate: true,
		},
		{
			name: "EnvVar exists with the same value and should not be updated",
			initialEnvVars: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
			envVarName:  "EXISTING_VAR",
			envVarValue: "existing_value",
			expectedEnv: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing_value"},
			},
			expectedUpdate: false,
		},
		{
			name: "EnvVar exists with a different value and should be updated",
			initialEnvVars: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "old_value"},
			},
			envVarName:  "EXISTING_VAR",
			envVarValue: "new_value",
			expectedEnv: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "new_value"},
			},
			expectedUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := createDeployment(tt.initialEnvVars)
			updated := updateDeploymentEnvVar(deployment, tt.envVarName, tt.envVarValue)

			if updated != tt.expectedUpdate {
				t.Errorf("expected update status to be %v, but got %v", tt.expectedUpdate, updated)
			}
			if diff := cmp.Diff(tt.expectedEnv, deployment.Spec.Template.Spec.Containers[0].Env); diff != "" {
				t.Errorf("deployment env mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateAuthorizationModelIdOnDeployment(t *testing.T) {
	reconcileTimestamp := time.Now()
	reconcileTimestampFormatted := reconcileTimestamp.UTC().Format(time.RFC3339)
	logger := log.FromContext(context.Background())

	tests := []struct {
		name               string
		deployments        appsV1.DeploymentList
		updates            map[DeploymentIdentifier]appsV1.Deployment
		authorizationModel *MockAuthorizationModel
		expectedUpdates    map[DeploymentIdentifier]appsV1.Deployment
	}{
		{
			name: "No deployments",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{},
			},
			updates:            map[DeploymentIdentifier]appsV1.Deployment{},
			authorizationModel: &MockAuthorizationModel{},
			expectedUpdates:    map[DeploymentIdentifier]appsV1.Deployment{},
		},
		{
			name: "Deployment with error in GetVersionFromDeployment",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations("namespace1", "error-deployment", []corev1.EnvVar{}, map[string]string{}),
				},
			},
			updates:            map[DeploymentIdentifier]appsV1.Deployment{},
			authorizationModel: &MockAuthorizationModel{},
			expectedUpdates:    map[DeploymentIdentifier]appsV1.Deployment{},
		},
		{
			name: "Deployment without the target env var",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations("namespace1", "deployment1", []corev1.EnvVar{}, map[string]string{}),
				},
			},
			updates:            map[DeploymentIdentifier]appsV1.Deployment{},
			authorizationModel: &MockAuthorizationModel{},
			expectedUpdates: map[DeploymentIdentifier]appsV1.Deployment{
				{namespace: "namespace1", name: "deployment1"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"deployment1",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: "auth-model-id"},
					},
					map[string]string{
						extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation: reconcileTimestampFormatted,
						extensionsv1.OpenFgaAuthModelVersionLabel:     "1.2.3",
					},
				),
			},
		},
		{
			name: "Deployment with the target env var already set to correct value",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations(
						"namespace1",
						"deployment1",
						[]corev1.EnvVar{
							{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: "auth-model-id"},
						},
						map[string]string{},
					),
				},
			},
			updates:            map[DeploymentIdentifier]appsV1.Deployment{},
			authorizationModel: &MockAuthorizationModel{},
			expectedUpdates:    map[DeploymentIdentifier]appsV1.Deployment{},
		},
		{
			name: "Pre-existing updates map with a deployment",
			deployments: appsV1.DeploymentList{
				Items: []appsV1.Deployment{
					createDeploymentWithNameAndAnnotations("namespace1", "deployment1", []corev1.EnvVar{}, map[string]string{}),
				},
			},
			updates: map[DeploymentIdentifier]appsV1.Deployment{
				{namespace: "namespace1", name: "existing-deployment"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"existing-deployment",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: "existing-value"},
					},
					map[string]string{
						extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation: "existing-timestamp",
					},
				),
			},
			authorizationModel: &MockAuthorizationModel{},
			expectedUpdates: map[DeploymentIdentifier]appsV1.Deployment{
				{namespace: "namespace1", name: "existing-deployment"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"existing-deployment",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: "existing-value"},
					},
					map[string]string{
						extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation: "existing-timestamp",
					},
				),
				{namespace: "namespace1", name: "deployment1"}: createDeploymentWithNameAndAnnotations(
					"namespace1",
					"deployment1",
					[]corev1.EnvVar{
						{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: "auth-model-id"},
					},
					map[string]string{
						extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation: reconcileTimestampFormatted,
						extensionsv1.OpenFgaAuthModelVersionLabel:     "1.2.3",
					},
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateAuthorizationModelIdOnDeployment(tt.deployments, tt.updates, tt.authorizationModel, reconcileTimestamp, &logger)

			if diff := cmp.Diff(tt.expectedUpdates, tt.updates); diff != "" {
				t.Errorf("unexpected updates (-want +got):\n%s", diff)
			}
		})
	}
}
