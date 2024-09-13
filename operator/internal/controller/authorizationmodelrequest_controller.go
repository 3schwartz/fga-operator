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
	"fga-operator/internal/observability"
	"fga-operator/internal/openfga"
	"fmt"
	"github.com/go-logr/logr"
	appsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	extensionsv1 "fga-operator/api/v1"
)

// AuthorizationModelRequestReconciler reconciles a AuthorizationModelRequest object
type AuthorizationModelRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	openfga.PermissionServiceFactory
	openfga.Config
	Clock
	RequeueAfter *time.Duration
}

type Clock interface {
	Now() time.Time
}

const deploymentIndexKey = ".metadata.labels." + extensionsv1.OpenFgaStoreLabel

//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciliation triggered")
	reconcileTimestamp := r.Now()

	requeueResult := ctrl.Result{RequeueAfter: *r.RequeueAfter}

	authorizationRequest := &extensionsv1.AuthorizationModelRequest{}
	if err := r.Get(ctx, req.NamespacedName, authorizationRequest); err != nil {
		logger.Error(err, "unable to fetch authorization model request", "authorizationModelRequestName", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	openFgaService, err := r.PermissionServiceFactory.GetService(r.Config)
	if err != nil {
		return ctrl.Result{}, err
	}

	store, err := r.getStore(ctx, req, openFgaService, authorizationRequest, &logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	authorizationModel, err := r.getAuthorizationModel(ctx, req, openFgaService, authorizationRequest, reconcileTimestamp, &logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err = r.updateAuthorizationModel(ctx, openFgaService, authorizationRequest, authorizationModel, reconcileTimestamp, &logger); err != nil {
		return ctrl.Result{}, err
	}

	var deployments appsV1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingFields{deploymentIndexKey: store.Name}); err != nil {
		logger.Error(err, "unable to list deployments")
		return ctrl.Result{}, err
	}

	updates := updateStoreIdOnDeployments(deployments, store, reconcileTimestamp)

	updateAuthorizationModelIdOnDeployment(deployments, updates, authorizationModel, reconcileTimestamp, &logger)

	for _, deployment := range updates {
		r.updateDeployment(ctx, &deployment, req.Name, &logger)
	}

	return requeueResult, nil
}

func (r *AuthorizationModelRequestReconciler) updateDeployment(
	ctx context.Context,
	deployment *appsV1.Deployment,
	modelName string,
	log *logr.Logger,

) {
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "unable to update deployment", "deploymentName", deployment.Name)
		return
	}
	observability.RecordDeploymentUpdated(deployment.Name, modelName)
	log.V(0).Info("deployment updated", "deploymentName", deployment.Name)
}

func updateAuthorizationModelWithMissingInstances(
	ctx context.Context,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	authorizationModel *extensionsv1.AuthorizationModel,
	reconcileTimestamp time.Time,
	log *logr.Logger) (bool, error) {
	missingInstances := make([]extensionsv1.AuthorizationModelRequestInstance, 0)
	existingVersions := make(map[extensionsv1.ModelVersion]struct{})

	for _, modelExisting := range authorizationModel.Spec.Instances {
		existingVersions[modelExisting.Version] = struct{}{}
	}

	for _, modelRequest := range authorizationModelRequest.Spec.Instances {
		if _, exists := existingVersions[modelRequest.Version]; !exists {
			missingInstances = append(missingInstances, modelRequest)
		}
	}

	if len(missingInstances) == 0 {
		return false, nil
	}

	modelInstances := authorizationModel.Spec.Instances
	for _, modelRequestInstance := range missingInstances {
		authModelId, err := openFgaService.CreateAuthorizationModel(ctx, modelRequestInstance.AuthorizationModel, log)
		if err != nil {
			return false, err
		}
		log.V(0).Info("Created new authorization model in OpenFGA",
			"authModel", authorizationModel.Name,
			"version", modelRequestInstance.Version.String(),
			"authModelId", authModelId)
		log.V(0).Info(fmt.Sprintf("Authorization model resource will updates it's instances with id: %s", authModelId),
			"authModel", authorizationModel.Name,
			"version", modelRequestInstance.Version.String(),
			"authModelId", authModelId)
		modelInstances = append(modelInstances, extensionsv1.AuthorizationModelInstance{
			Id:                 authModelId,
			AuthorizationModel: modelRequestInstance.AuthorizationModel,
			Version:            modelRequestInstance.Version,
			CreatedAt:          &metav1.Time{Time: reconcileTimestamp},
		})
	}

	authorizationModel.Spec.Instances = modelInstances
	return true, nil
}

func removeObsoleteInstances(
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	authorizationModel *extensionsv1.AuthorizationModel,
	log *logr.Logger) bool {

	requestedModelVersions := make(map[extensionsv1.ModelVersion]struct{})

	for _, requestModel := range authorizationModelRequest.Spec.Instances {
		requestedModelVersions[requestModel.Version] = struct{}{}
	}

	existingInstances := make([]extensionsv1.AuthorizationModelInstance, 0)
	for _, existingModel := range authorizationModel.Spec.Instances {
		if _, exists := requestedModelVersions[existingModel.Version]; !exists {
			log.V(0).Info(fmt.Sprintf("Authorization model resource will remove the instance with id: %s", existingModel.Id),
				"authModel", authorizationModel.Name,
				"version", existingModel.Version,
				"authModelId", existingModel.Id)
			continue
		}
		existingInstances = append(existingInstances, existingModel)
	}
	if len(existingInstances) == len(authorizationModel.Spec.Instances) {
		return false
	}
	authorizationModel.Spec.Instances = existingInstances
	return true
}

func (r *AuthorizationModelRequestReconciler) updateAuthorizationModel(
	ctx context.Context,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	authorizationModel *extensionsv1.AuthorizationModel,
	reconcileTimestamp time.Time,
	log *logr.Logger) error {

	updateMissing, err := updateAuthorizationModelWithMissingInstances(ctx, openFgaService, authorizationModelRequest, authorizationModel, reconcileTimestamp, log)
	if err != nil {
		return err
	}
	removeObsolete := removeObsoleteInstances(authorizationModelRequest, authorizationModel, log)

	if !(updateMissing || removeObsolete) {
		return nil
	}

	if err := r.Update(ctx, authorizationModel); err != nil {
		log.Error(err, "unable to update authorization model in Kubernetes", "authorizationModel", authorizationModel)
		return err
	}
	observability.RecordK8AuthorizationModelEvent(observability.Updated, authorizationModel.Name)
	log.V(0).Info("Updated authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return nil
}

func (r *AuthorizationModelRequestReconciler) getAuthorizationModel(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	reconcileTimestamp time.Time,
	log *logr.Logger) (*extensionsv1.AuthorizationModel, error) {

	authorizationModel := &extensionsv1.AuthorizationModel{}
	err := r.Get(ctx, req.NamespacedName, authorizationModel)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return nil, err
	case errors.IsNotFound(err):
		authorizationModel, err = r.createAuthorizationModel(ctx, req, openFgaService, authorizationModelRequest, reconcileTimestamp, log)
		if err != nil {
			return nil, err
		}
	}
	return authorizationModel, nil
}

func (r *AuthorizationModelRequestReconciler) createAuthorizationModel(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	reconcileTimestamp time.Time,
	log *logr.Logger) (*extensionsv1.AuthorizationModel, error) {

	definitions := make([]extensionsv1.AuthorizationModelDefinition, len(authorizationModelRequest.Spec.Instances))
	for i, instance := range authorizationModelRequest.Spec.Instances {
		authModelId, err := openFgaService.CreateAuthorizationModel(ctx, instance.AuthorizationModel, log)
		if err != nil {
			return nil, err
		}
		observability.RecordOpenFgaAuthorizationModels(req.Name)

		definitions[i] = extensionsv1.NewAuthorizationModelDefinition(authModelId, instance.AuthorizationModel, instance.Version)
	}

	authorizationModel := extensionsv1.NewAuthorizationModel(req.Name, req.Namespace, definitions, reconcileTimestamp)

	if err := ctrl.SetControllerReference(authorizationModelRequest, &authorizationModel, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, &authorizationModel); err != nil {
		log.Error(err, fmt.Sprintf("Failed to create authorization model %s", req.Name))
		return nil, err
	}
	observability.RecordK8AuthorizationModelEvent(observability.Created, req.Name)
	log.V(0).Info("Created authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return &authorizationModel, nil
}

func (r *AuthorizationModelRequestReconciler) getStore(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.Store, error) {

	store := &extensionsv1.Store{}
	err := r.Get(ctx, req.NamespacedName, store)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return nil, err
	case errors.IsNotFound(err):
		store, err = r.createStoreResource(ctx, req, openFgaService, authorizationModelRequest, log)
		if err != nil {
			return nil, err
		}
	}
	openFgaService.SetStoreId(store.Spec.Id)
	return store, nil
}

func (r *AuthorizationModelRequestReconciler) createStoreResource(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.Store, error) {

	store, err := openFgaService.CheckExistingStores(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if store == nil {
		store, err = openFgaService.CreateStore(ctx, req.Name, log)
		observability.RecordOpenFgaStoreEvent(req.Name)
		if err != nil {
			return nil, err
		}
	}
	storeResource := extensionsv1.NewStore(store.Name, req.Namespace, store.Id, store.CreatedAt)

	if err := ctrl.SetControllerReference(authorizationModelRequest, storeResource, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, storeResource); client.IgnoreAlreadyExists(err) != nil {
		log.Error(err, fmt.Sprintf("Failed to create store %s", req.Name))
		return nil, err
	}
	observability.RecordK8StoreEvent(req.Name)
	log.V(0).Info("Created store in Kubernetes", "storeKubernetes", storeResource)

	return storeResource, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationModelRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsV1.Deployment{}, deploymentIndexKey, func(rawObj client.Object) []string {
		deployment := rawObj.(*appsV1.Deployment)
		labelValue, exists := deployment.Labels[extensionsv1.OpenFgaStoreLabel]
		if !exists {
			return nil
		}
		return []string{labelValue}
	}); err != nil {
		return err
	}

	deletePredicate := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			if object, ok := e.Object.(*extensionsv1.AuthorizationModelRequest); ok {
				observability.RecordK8AuthorizationModelEvent(observability.Deleted, object.Name)
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1.AuthorizationModelRequest{}).
		WithEventFilter(deletePredicate).
		Complete(r)
}
