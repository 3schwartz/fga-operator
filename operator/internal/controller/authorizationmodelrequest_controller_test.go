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

package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	extensionsv1 "openfga-controller/api/v1"
	openfgainternal "openfga-controller/internal/openfga"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	model = `
model
  schema 1.1

type user

type document
  relations
    define foo: [user]
    define reader: [user]
    define writer: [user]
    define owner: [user]
`
	modelUpdated = `
model
  schema 1.1

type user
  relations
    define owner: [user]

type document
  relations
    define member: [user]
`
	version        = "1.1.1"
	versionUpdated = "1.1.2"
	duration       = time.Second * 3
	interval       = time.Millisecond * 250
)

func createAuthorizationModelRequestWithSpecs(name, namespace, version, model string) extensionsv1.AuthorizationModelRequest {
	return extensionsv1.AuthorizationModelRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(uuid.NewString()),
		},
		Spec: extensionsv1.AuthorizationModelRequestSpec{
			Version:            version,
			AuthorizationModel: model,
		},
	}
}

func createAuthorizationModelRequest(name, namespace string) extensionsv1.AuthorizationModelRequest {
	return createAuthorizationModelRequestWithSpecs(name, namespace, version, model)
}

func createAuthorizationModel(name, namespace string) extensionsv1.AuthorizationModel {
	return extensionsv1.NewAuthorizationModel(name, namespace, uuid.NewString(), version, model, time.Now())
}

var _ = Describe("AuthorizationModelRequest Controller", func() {
	Context("When reconciling a resource", func() {
		logger := log.FromContext(ctx)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespaceName,
		}
		request := reconcile.Request{NamespacedName: typeNamespacedName}

		deleteResource := func(resource client.Object) {
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
			if errors.IsNotFound(err) {
				return
			}
			resourceType := reflect.TypeOf(resource).Elem().Name()
			By(fmt.Sprintf("Cleanup the specific resource instance %s", resourceType))
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		}

		AfterEach(func() {
			deleteResource(&extensionsv1.AuthorizationModelRequest{})
			deleteResource(&extensionsv1.AuthorizationModel{})
			deleteResource(&extensionsv1.Store{})
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			// Arrange
			authorizationModelRequest := &extensionsv1.AuthorizationModelRequest{}
			err := k8sClient.Get(ctx, typeNamespacedName, authorizationModelRequest)
			if err != nil && errors.IsNotFound(err) {
				resource := createAuthorizationModelRequest(resourceName, namespaceName)
				Expect(k8sClient.Create(ctx, &resource)).To(Succeed())
			}

			Eventually(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).Should(Succeed())
		})

		It("given existing store when create store resource then return existing", func() {
			mockService := openfgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().CheckExistingStores(gomock.Any(), gomock.Any()).Return(&openfgainternal.Store{
				Id:        "foo",
				Name:      resourceName,
				CreatedAt: time.Now(),
			}, nil)

			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)

			storeResource, err := controllerReconciler.createStoreResource(
				ctx, request,
				mockService, &authRequest, &logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(storeResource).NotTo(BeNil())
			Expect(storeResource.Name).To(Equal(resourceName))
			Expect(storeResource.Namespace).To(Equal(namespaceName))
		})

		It("given no existing store when create store resource then create new store", func() {
			storeId := uuid.NewString()
			mockService := openfgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().CheckExistingStores(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(&openfgainternal.Store{
				Id:        storeId,
				Name:      resourceName,
				CreatedAt: time.Now(),
			}, nil)

			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)

			storeResource, err := controllerReconciler.createStoreResource(
				ctx, request,
				mockService, &authRequest, &logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(storeResource).NotTo(BeNil())
			Expect(storeResource.Spec.Id).To(Equal(storeId))
			Expect(storeResource.Name).To(Equal(resourceName))
			Expect(storeResource.Namespace).To(Equal(namespaceName))
		})

		It("when create authorization model then present in kubernetes", func() {
			// Arrange
			authModelId := uuid.NewString()
			mockService := openfgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authModelId, nil)
			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)

			// Act
			authModel, err := controllerReconciler.createAuthorizationModel(ctx, request, mockService, &authRequest, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(authModel).NotTo(BeNil())
			Expect(authModel.Spec.Instance.Id).To(Equal(authModelId))
			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(authModelInK8.Spec.Instance.Id).To(Equal(authModelId))
		})

		It("given no changes in auth model when update then do not changes", func() {
			// Arrange
			mockService := openfgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)
			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)
			authModel := createAuthorizationModel(resourceName, namespaceName)

			// Act
			err := controllerReconciler.updateAuthorizationModel(ctx, mockService, &authRequest, &authModel, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
		})

		It("given changes in auth model when update then do changes", func() {
			// Arrange
			authModel := createAuthorizationModel(resourceName, namespaceName)
			Expect(k8sClient.Create(ctx, &authModel)).To(Succeed())
			authModelRequest := createAuthorizationModelRequestWithSpecs(resourceName, namespaceName, versionUpdated, modelUpdated)
			oldAuthModelId := authModel.Spec.Instance.Id
			newAuthModelId := uuid.NewString()
			mockService := openfgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(newAuthModelId, nil)

			Expect(len(authModel.Spec.LatestModels)).To(Equal(0))

			// Act
			err := controllerReconciler.updateAuthorizationModel(ctx, mockService, &authModelRequest, &authModel, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(len(authModel.Spec.LatestModels)).To(Equal(1))
			Expect(authModel.Spec.LatestModels[0].Id).To(Equal(oldAuthModelId))
			Expect(authModel.Spec.Instance.Id).To(Equal(newAuthModelId))
			Expect(authModel.Spec.Instance.Version).To(Equal(versionUpdated))
			Expect(authModel.Spec.AuthorizationModel).To(Equal(modelUpdated))
			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(authModelInK8.Spec.Instance.Id).To(Equal(newAuthModelId))
			Expect(authModelInK8.Spec.Instance.Version).To(Equal(versionUpdated))
			Expect(authModelInK8.Spec.AuthorizationModel).To(Equal(modelUpdated))
			Expect(len(authModelInK8.Spec.LatestModels)).To(Equal(1))
		})
	})
})
