/*
Copyright 2024.

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

package authorizationmodel

import (
	extensionsv1 "fga-operator/api/v1"
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"time"
)

const (
	duration = time.Second * 3
	interval = time.Millisecond * 250
)

func getLowercaseUUID() string {
	return "a" + strings.ToLower(uuid.NewString())
}

var _ = Describe("AuthorizationModel Controller", func() {
	Context("When reconciling a resource", func() {
		var storeId string
		var name string

		BeforeEach(func() {
			eventRecorder.Events = make(chan string, 5)
			name = getLowercaseUUID()
			storeId = getLowercaseUUID()
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: name,
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			store := extensionsv1.Store{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: name,
				},
				Spec: extensionsv1.StoreSpec{
					Id: storeId,
				},
			}
			Expect(k8sClient.Create(ctx, &store)).To(Succeed())
		})

		It("given auth model version label then update deployment to correct version", func() {
			// Arrange
			authModelId := getLowercaseUUID()
			deploymentName := getLowercaseUUID()
			modelVersion := extensionsv1.ModelVersion{
				Major: 0,
				Minor: 0,
				Patch: 1,
			}

			deployment := createDeploymentWithAnnotations(name, deploymentName, map[string]string{
				extensionsv1.OpenFgaStoreLabel:            name,
				extensionsv1.OpenFgaAuthModelVersionLabel: modelVersion.String(),
			})
			Expect(k8sClient.Create(ctx, &deployment)).To(Succeed())

			authorizationModel := extensionsv1.AuthorizationModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: name,
				},
				Spec: extensionsv1.AuthorizationModelSpec{
					Instances: []extensionsv1.AuthorizationModelInstance{
						{
							Id: getLowercaseUUID(),
							Version: extensionsv1.ModelVersion{
								Major: 1,
								Minor: 2,
								Patch: 3,
							},
							AuthorizationModel: getLowercaseUUID(),
						},
						{
							Id:                 authModelId,
							Version:            modelVersion,
							AuthorizationModel: getLowercaseUUID(),
						},
					},
				},
			}

			// Act
			Expect(k8sClient.Create(ctx, &authorizationModel)).To(Succeed())

			// Assert
			validateDeployment(deploymentName, name, storeId, authModelId, modelVersion)
			validateNoEventsFound(eventRecorder.Events)
		})

		It("given no auth model version label then update deployment to latest", func() {
			// Arrange
			authModelId := getLowercaseUUID()
			deploymentName := getLowercaseUUID()

			deployment := createDeploymentWithAnnotations(name, deploymentName, map[string]string{
				extensionsv1.OpenFgaStoreLabel: name,
			})
			Expect(k8sClient.Create(ctx, &deployment)).To(Succeed())

			modelVersion := extensionsv1.ModelVersion{
				Major: 1,
				Minor: 2,
				Patch: 3,
			}
			authorizationModel := extensionsv1.AuthorizationModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: name,
				},
				Spec: extensionsv1.AuthorizationModelSpec{
					Instances: []extensionsv1.AuthorizationModelInstance{
						{
							Id:                 authModelId,
							Version:            modelVersion,
							AuthorizationModel: getLowercaseUUID(),
						},
						{
							Id: getLowercaseUUID(),
							Version: extensionsv1.ModelVersion{
								Major: 0,
								Minor: 0,
								Patch: 1,
							},
							AuthorizationModel: getLowercaseUUID(),
						},
					},
				},
			}

			// Act
			Expect(k8sClient.Create(ctx, &authorizationModel)).To(Succeed())

			// Assert
			validateDeployment(deploymentName, name, storeId, authModelId, modelVersion)
			validateNoEventsFound(eventRecorder.Events)
		})
	})
})

func createDeploymentWithAnnotations(namespaceName, deploymentName string, annotations map[string]string) appsV1.Deployment {
	return appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      deploymentName,
			Labels:    annotations,
		},
		Spec: appsV1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploymentName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Name: getLowercaseUUID(), Labels: map[string]string{"app": deploymentName}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: getLowercaseUUID(),
							Name:  getLowercaseUUID(),
						},
						{
							Image: getLowercaseUUID(),
							Name:  getLowercaseUUID(),
							Env: []corev1.EnvVar{
								{Name: extensionsv1.OpenFgaAuthModelIdEnv, Value: getLowercaseUUID()},
							},
						},
						{
							Image: getLowercaseUUID(),
							Name:  getLowercaseUUID(),
							Env: []corev1.EnvVar{
								{Name: extensionsv1.OpenFgaStoreIdEnv, Value: getLowercaseUUID()},
							},
						},
					},
				},
			},
		},
	}
}

func validateDeployment(deploymentName, namespaceName, storeId, authModelId string, modelVersion extensionsv1.ModelVersion) {
	Eventually(func() error {
		deployment := &appsV1.Deployment{}
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      deploymentName,
			Namespace: namespaceName,
		}, deployment); err != nil {
			return err
		}

		// Validate that all containers have the necessary environment variables
		for _, container := range deployment.Spec.Template.Spec.Containers {
			foundStoreId := false
			foundAuthModelId := false

			// Check environment variables in each container
			for _, envVar := range container.Env {
				if envVar.Name == extensionsv1.OpenFgaStoreIdEnv && envVar.Value == storeId {
					foundStoreId = true
				}
				if envVar.Name == extensionsv1.OpenFgaAuthModelIdEnv && envVar.Value == authModelId {
					foundAuthModelId = true
				}
			}

			if !foundStoreId {
				return fmt.Errorf("container %s does not have env var %s with value %s", container.Name, extensionsv1.OpenFgaStoreIdEnv, storeId)
			}
			if !foundAuthModelId {
				return fmt.Errorf("container %s does not have env var %s with value %s", container.Name, extensionsv1.OpenFgaAuthModelIdEnv, authModelId)
			}
		}

		// Validate that the deployment has the annotation OpenFgaAuthModel with the correct version
		if deployment.Annotations[extensionsv1.OpenFgaAuthModelVersionLabel] != modelVersion.String() {
			return fmt.Errorf("deployment does not have annotation %s with value %s", extensionsv1.OpenFgaAuthModelVersionLabel, modelVersion.String())
		}
		if deployment.Annotations[extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation] != MockTimeAsString() {
			return fmt.Errorf("deployment does not have annotation %s with value %s", extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation, MockTimeAsString())
		}
		if deployment.Annotations[extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation] != MockTimeAsString() {
			return fmt.Errorf("deployment does not have annotation %s with value %s", extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation, MockTimeAsString())
		}

		return nil
	}, duration, interval).Should(Succeed())
}

func validateNoEventsFound(events <-chan string) {
	Consistently(func() bool {
		select {
		case <-events:
			return false // Event received or channel closed
		default:
			return true // No event received
		}
	}, duration, interval).Should(BeTrue())
}
