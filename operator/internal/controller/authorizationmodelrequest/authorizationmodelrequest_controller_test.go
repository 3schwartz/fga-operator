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

package authorizationmodelrequest

import (
	"context"
	extensionsv1 "fga-operator/api/v1"
	fgainternal "fga-operator/internal/openfga"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
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
	duration = time.Millisecond * 1_500
	interval = time.Millisecond * 250
)

var (
	version = extensionsv1.ModelVersion{
		Major: 1,
		Minor: 1,
		Patch: 1,
	}
	versionUpdated = extensionsv1.ModelVersion{
		Major: 1,
		Minor: 1,
		Patch: 2,
	}
)

func authorizationModelRequestInstancesFromSingle(model string, version extensionsv1.ModelVersion) []extensionsv1.AuthorizationModelRequestInstance {
	return []extensionsv1.AuthorizationModelRequestInstance{
		{
			AuthorizationModel: model,
			Version:            version,
		},
	}
}

func createAuthorizationModelRequestWithSpecs(name, namespace string, instances []extensionsv1.AuthorizationModelRequestInstance) extensionsv1.AuthorizationModelRequest {
	return extensionsv1.AuthorizationModelRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(uuid.NewString()),
		},
		Spec: extensionsv1.AuthorizationModelRequestSpec{
			Instances: instances,
		},
	}
}

func createAuthorizationModelRequest(name, namespace string) extensionsv1.AuthorizationModelRequest {
	return createAuthorizationModelRequestWithSpecs(name, namespace, authorizationModelRequestInstancesFromSingle(model, version))
}

func createAuthorizationModel(name, namespace string) extensionsv1.AuthorizationModel {
	definition := extensionsv1.NewAuthorizationModelDefinition(uuid.NewString(), model, version)
	return extensionsv1.NewAuthorizationModel(name, namespace, []extensionsv1.AuthorizationModelDefinition{definition}, time.Now())
}

func ensureAuthorizationModelRequestExists(ctx context.Context, typeNamespacedName types.NamespacedName) {
	authorizationModelRequest := &extensionsv1.AuthorizationModelRequest{}
	err := k8sClient.Get(ctx, typeNamespacedName, authorizationModelRequest)
	if err != nil && errors.IsNotFound(err) {
		resource := createAuthorizationModelRequest(resourceName, namespaceName)
		Expect(k8sClient.Create(ctx, &resource)).To(Succeed())
	}
}

var _ = Describe("AuthorizationModelRequest Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()
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

		It("given existing store but missing in open fga, then do not create store resource", func() {
			// Arrange
			existingStoreId := uuid.NewString()
			existingAuthorizationModelId := uuid.NewString()
			resource := extensionsv1.AuthorizationModelRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
					UID:       types.UID(uuid.NewString()),
				},
				Spec: extensionsv1.AuthorizationModelRequestSpec{
					ExistingStoreId: existingStoreId,
					Instances: []extensionsv1.AuthorizationModelRequestInstance{
						{
							ExistingAuthorizationModelId: existingAuthorizationModelId,
							AuthorizationModel:           model,
							Version:                      version,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &resource)).To(Succeed())

			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil).Times(1)

			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("store not found")).Times(1)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().SetStoreId(gomock.Any()).Times(0)

			mockService.EXPECT().CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckAuthorizationModelExists(gomock.Any(), gomock.Any()).Times(0)

			fakeRecorder := record.NewFakeRecorder(5)
			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 fakeRecorder,
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)

			// Assert
			Expect(err).To(HaveOccurred())
			Consistently(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).ShouldNot(Succeed())
			Consistently(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).ShouldNot(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.SynchronizationFailed))
			validateEvent(fakeRecorder.Events, EventReasonStoreFailed)
		})

		It("given existing authorization model but missing in open fga, then do not create authorization model resource", func() {
			// Arrange
			existingStoreId := uuid.NewString()
			existingAuthorizationModelId := uuid.NewString()
			resource := extensionsv1.AuthorizationModelRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
					UID:       types.UID(uuid.NewString()),
				},
				Spec: extensionsv1.AuthorizationModelRequestSpec{
					ExistingStoreId: existingStoreId,
					Instances: []extensionsv1.AuthorizationModelRequestInstance{
						{
							ExistingAuthorizationModelId: existingAuthorizationModelId,
							AuthorizationModel:           model,
							Version:                      version,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &resource)).To(Succeed())

			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil).Times(1)

			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Return(&fgainternal.Store{
				Id:        existingStoreId,
				Name:      resourceName,
				CreatedAt: time.Now(),
			}, nil).Times(1)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().SetStoreId(gomock.Any()).Times(1)

			mockService.EXPECT().CheckAuthorizationModelExists(gomock.Any(), gomock.Any()).Times(1).Return(false, nil)
			mockService.EXPECT().CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			fakeRecorder := record.NewFakeRecorder(5)
			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 fakeRecorder,
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)

			// Assert
			Expect(err).To(HaveOccurred())
			Eventually(func() error {
				store := &extensionsv1.Store{}
				err := k8sClient.Get(ctx, typeNamespacedName, store)
				if err != nil {
					return err
				}
				if store.Spec.Id != existingStoreId {
					return fmt.Errorf("expected store id %s, got %s", existingStoreId, store.Spec.Id)
				}
				return nil
			}, duration, interval).Should(Succeed())
			Consistently(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).ShouldNot(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.SynchronizationFailed))
			validateEvent(fakeRecorder.Events, EventReasonAuthorizationModelCreationFailed)
		})

		It("given existing store and authorization model, then do not call open fga", func() {
			// Arrange
			existingStoreId := uuid.NewString()
			existingAuthorizationModelId := uuid.NewString()
			resource := extensionsv1.AuthorizationModelRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespaceName,
					UID:       types.UID(uuid.NewString()),
				},
				Spec: extensionsv1.AuthorizationModelRequestSpec{
					ExistingStoreId: existingStoreId,
					Instances: []extensionsv1.AuthorizationModelRequestInstance{
						{
							ExistingAuthorizationModelId: existingAuthorizationModelId,
							AuthorizationModel:           model,
							Version:                      version,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &resource)).To(Succeed())

			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil).Times(1)

			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Return(&fgainternal.Store{
				Id:        existingStoreId,
				Name:      resourceName,
				CreatedAt: time.Now(),
			}, nil).Times(1)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().SetStoreId(gomock.Any()).Times(1)

			mockService.EXPECT().CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckAuthorizationModelExists(gomock.Any(), gomock.Any()).Times(1).Return(true, nil)

			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 record.NewFakeRecorder(5),
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)

			// Assert
			Expect(err).To(BeNil())
			Eventually(func() error {
				store := &extensionsv1.Store{}
				err := k8sClient.Get(ctx, typeNamespacedName, store)
				if err != nil {
					return err
				}
				if store.Spec.Id != existingStoreId {
					return fmt.Errorf("expected store id %s, got %s", existingStoreId, store.Spec.Id)
				}
				return nil
			}, duration, interval).Should(Succeed())
			Eventually(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				err := k8sClient.Get(ctx, typeNamespacedName, authModel)
				if err != nil {
					return err
				}
				if len(authModel.Spec.Instances) != 1 {
					return fmt.Errorf("expected 1 instance, got %d", len(authModel.Spec.Instances))
				}
				if authModel.Spec.Instances[0].Id != existingAuthorizationModelId {
					return fmt.Errorf("expected authorization model id %s, got %s", existingAuthorizationModelId, authModel.Spec.Instances[0].Id)
				}
				return nil
			}, duration, interval).Should(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.Synchronized))
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			// Arrange
			ensureAuthorizationModelRequestExists(ctx, typeNamespacedName)

			// Act
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).To(Not(HaveOccurred()))

			// Assert
			Eventually(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).Should(Succeed())
			Eventually(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).Should(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.Synchronized))
		})

		It("should show pending when not reconciled", func() {
			// Arrange
			ensureAuthorizationModelRequestExists(ctx, typeNamespacedName)

			// Assert
			Consistently(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.Pending))
		})

		It("should have status synchronization failed, when not able to get service", func() {
			// Arrange
			ensureAuthorizationModelRequestExists(ctx, typeNamespacedName)

			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(nil, fmt.Errorf("error"))

			fakeRecorder := record.NewFakeRecorder(5)
			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 fakeRecorder,
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)

			// Assert
			Expect(err).To(HaveOccurred())
			Consistently(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).ShouldNot(Succeed())
			Consistently(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).ShouldNot(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.SynchronizationFailed))
			validateEvent(fakeRecorder.Events, EventReasonClientInitializationFailed)
		})

		It("should have status synchronization failed, when not able to create store", func() {
			ensureAuthorizationModelRequestExists(ctx, typeNamespacedName)

			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil)
			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))

			fakeRecorder := record.NewFakeRecorder(5)
			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 fakeRecorder,
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Consistently(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).ShouldNot(Succeed())
			Consistently(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).ShouldNot(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.SynchronizationFailed))
			validateEvent(fakeRecorder.Events, EventReasonStoreFailed)
		})

		It("should have status synchronization failed, when not able to create authorization model", func() {
			ensureAuthorizationModelRequestExists(ctx, typeNamespacedName)
			store := fgainternal.Store{
				Id:        "foo",
				Name:      resourceName,
				CreatedAt: time.Now(),
			}
			mockFactory := fgainternal.NewMockPermissionServiceFactory(goMockController)
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockFactory.EXPECT().GetService(gomock.Any()).Return(mockService, nil)
			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(&store, nil)
			mockService.EXPECT().SetStoreId(gomock.Any())
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return("", fmt.Errorf("error"))

			fakeRecorder := record.NewFakeRecorder(5)
			reconciler := &AuthorizationModelRequestReconciler{
				Client:                   k8sClient,
				Scheme:                   k8sClient.Scheme(),
				Recorder:                 fakeRecorder,
				Clock:                    clock.RealClock{},
				PermissionServiceFactory: mockFactory,
			}

			// Act
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Eventually(func() error {
				store := &extensionsv1.Store{}
				return k8sClient.Get(ctx, typeNamespacedName, store)
			}, duration, interval).Should(Succeed())
			Consistently(func() error {
				authModel := &extensionsv1.AuthorizationModel{}
				return k8sClient.Get(ctx, typeNamespacedName, authModel)
			}, duration, interval).ShouldNot(Succeed())
			Eventually(func() (extensionsv1.AuthorizationModelRequestStatusState, error) {
				authModelRequest := &extensionsv1.AuthorizationModelRequest{}
				if err := k8sClient.Get(ctx, typeNamespacedName, authModelRequest); err != nil {
					return "", err
				}
				return authModelRequest.Status.State, nil
			}, duration, interval).Should(Equal(extensionsv1.SynchronizationFailed))
			validateEvent(fakeRecorder.Events, EventReasonAuthorizationModelCreationFailed)
		})

		It("given existing store when create store resource then return existing", func() {
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Return(&fgainternal.Store{
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
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().CheckExistingStoresByName(gomock.Any(), gomock.Any()).Return(nil, nil)
			mockService.EXPECT().CheckExistingStoresById(gomock.Any(), gomock.Any()).Times(0)
			mockService.EXPECT().CreateStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(&fgainternal.Store{
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
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authModelId, nil)
			authRequest := createAuthorizationModelRequest(resourceName, namespaceName)

			// Act
			authModel, err := controllerReconciler.createAuthorizationModel(ctx, request, mockService, &authRequest, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(authModel).NotTo(BeNil())
			Expect(len(authModel.Spec.Instances)).To(Equal(1))
			Expect(authModel.Spec.Instances[0].Id).To(Equal(authModelId))
			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(authModelInK8.Spec.Instances[0].Id).To(Equal(authModelId))
		})

		It("given no changes in auth model when update then do not changes", func() {
			// Arrange
			mockService := fgainternal.NewMockPermissionService(goMockController)
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
			requestInstances := []extensionsv1.AuthorizationModelRequestInstance{
				{
					AuthorizationModel: authModel.Spec.Instances[0].AuthorizationModel,
					Version:            authModel.Spec.Instances[0].Version,
				},
				{
					AuthorizationModel: modelUpdated,
					Version:            versionUpdated,
				},
			}
			Expect(k8sClient.Create(ctx, &authModel)).To(Succeed())
			authModelRequest := createAuthorizationModelRequestWithSpecs(resourceName, namespaceName, requestInstances)
			newAuthModelId := uuid.NewString()
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(newAuthModelId, nil)

			Expect(len(authModel.Spec.Instances)).To(Equal(1))
			oldAuthModelId := authModel.Spec.Instances[0].Id

			// Act
			err := controllerReconciler.updateAuthorizationModel(ctx, mockService, &authModelRequest, &authModel, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(len(authModel.Spec.Instances)).To(Equal(2))

			extensionsv1.SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(authModel.Spec.Instances)
			Expect(authModel.Spec.Instances[1].Id).To(Equal(oldAuthModelId))
			newModel := authModel.Spec.Instances[0]
			Expect(newModel.Id).To(Equal(newAuthModelId))
			Expect(newModel.Version).To(Equal(versionUpdated))
			Expect(newModel.AuthorizationModel).To(Equal(modelUpdated))

			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(len(authModelInK8.Spec.Instances)).To(Equal(2))
			extensionsv1.SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(authModelInK8.Spec.Instances)
			newModelK8 := authModelInK8.Spec.Instances[0]
			Expect(newModelK8.Id).To(Equal(newAuthModelId))
			Expect(newModelK8.Version).To(Equal(versionUpdated))
			Expect(newModelK8.AuthorizationModel).To(Equal(modelUpdated))
		})

		It("when remove model from request then remove model from auth model resource", func() {
			// Arrange
			authModel := createAuthorizationModel(resourceName, namespaceName)
			Expect(k8sClient.Create(ctx, &authModel)).To(Succeed())
			requestInstances := authorizationModelRequestInstancesFromSingle(modelUpdated, versionUpdated)
			authModelRequest := createAuthorizationModelRequestWithSpecs(resourceName, namespaceName, requestInstances)
			newAuthModelId := uuid.NewString()
			mockService := fgainternal.NewMockPermissionService(goMockController)
			mockService.EXPECT().
				CreateAuthorizationModel(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(newAuthModelId, nil)

			Expect(len(authModel.Spec.Instances)).To(Equal(1))

			// Act
			err := controllerReconciler.updateAuthorizationModel(ctx, mockService, &authModelRequest, &authModel, time.Now(), &logger)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(len(authModel.Spec.Instances)).To(Equal(1))
			instance := authModel.Spec.Instances[0]
			Expect(instance.Id).To(Equal(newAuthModelId))
			Expect(instance.Version).To(Equal(versionUpdated))
			Expect(instance.AuthorizationModel).To(Equal(modelUpdated))

			var authModelInK8 extensionsv1.AuthorizationModel
			Expect(k8sClient.Get(ctx, typeNamespacedName, &authModelInK8)).To(Succeed())
			Expect(len(authModelInK8.Spec.Instances)).To(Equal(1))
			instanceK8 := authModelInK8.Spec.Instances[0]
			Expect(instanceK8.Id).To(Equal(newAuthModelId))
			Expect(instanceK8.Version).To(Equal(versionUpdated))
			Expect(instanceK8.AuthorizationModel).To(Equal(modelUpdated))
		})
	})
})

func validateEvent(events <-chan string, eventReason EventReason) {
	select {
	case event := <-events:
		Expect(event).To(ContainSubstring(string(eventReason)))
	default:
		Fail("Expected an event, but no events were recorded")
	}
	// Ensure there are no additional events
	Consistently(func() string {
		select {
		case event := <-events:
			return event
		default:
			return ""
		}
	}, duration, interval).Should(BeEmpty())
}
