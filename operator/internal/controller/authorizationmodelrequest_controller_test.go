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
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	extensionsv1 "openfga-controller/api/v1"
	openfgainternal "openfga-controller/internal/openfga"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const model = `
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
const version = "1.1.1"

func createAuthorizationModelRequest(name, namespace string) extensionsv1.AuthorizationModelRequest {
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

var _ = Describe("AuthorizationModelRequest Controller", func() {
	Context("When reconciling a resource", func() {
		logger := log.FromContext(context.Background())
		const resourceName = "test-resource"
		const namespaceName = "default"
		var ctrl *gomock.Controller
		var mockFactory *openfgainternal.MockPermissionServiceFactory
		var controllerReconciler *AuthorizationModelRequestReconciler

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespaceName,
		}
		request := reconcile.Request{NamespacedName: typeNamespacedName}

		BeforeEach(func() {
			By("creating the custom resource for the Kind AuthorizationModelRequest")
			ctrl = gomock.NewController(GinkgoT())
			mockFactory = openfgainternal.NewMockPermissionServiceFactory(ctrl)
			controllerReconciler = &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				PermissionServiceFactory: mockFactory,
				Clock:                    clock.RealClock{},
			}
		})

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
			defer ctrl.Finish()

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

			mockService := openfgainternal.NewMockPermissionService(ctrl)
			store := openfgainternal.Store{
				Id:        "foo",
				Name:      "bar",
				CreatedAt: time.Now(),
			}
			authModelId := "123"

			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil)
			mockService.EXPECT().CheckExistingStores(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(&store, nil)
			mockService.EXPECT().SetStoreId(gomock.Any())
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authModelId, nil)
			mockService.EXPECT().SetAuthorizationModelId(gomock.Any()).Return(nil)

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})

		It("given existing store when create store resource then return existing", func() {
			mockService := openfgainternal.NewMockPermissionService(ctrl)
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
			mockService := openfgainternal.NewMockPermissionService(ctrl)
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
			mockService := openfgainternal.NewMockPermissionService(ctrl)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authModelId, nil)
			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)

			// Act
			authModel, err := controllerReconciler.createAuthorizationModel(ctx, request, mockService, &authRequest, &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(authModel).NotTo(BeNil())
			Expect(authModel.Spec.Instance.Id).To(Equal(authModelId))
			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(authModelInK8.Spec.Instance.Id).To(Equal(authModelId))
		})
	})
})
