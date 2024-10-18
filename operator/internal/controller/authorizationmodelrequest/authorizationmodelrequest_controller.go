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
	"fga-operator/internal/observability"
	"fga-operator/internal/openfga"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
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

const (
	EventRecorderLabel = "AuthorizationModelRequestReconciler"
)

type EventReason string

const (
	EventReasonAuthorizationModelStatusChangeFailed EventReason = "AuthorizationModelStatusChangeFailed"
	EventReasonClientInitializationFailed           EventReason = "ClientInitializationFailed"
	EventReasonStoreFailed                          EventReason = "StoreFailed"
	EventReasonAuthorizationModelCreationFailed     EventReason = "AuthorizationModelCreationFailed"
	EventReasonAuthorizationModelUpdateFailed       EventReason = "AuthorizationModelUpdateFailed"
)

// AuthorizationModelRequestReconciler reconciles a AuthorizationModelRequest object
type AuthorizationModelRequestReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	openfga.PermissionServiceFactory
	openfga.Config
	Clock
}

type Clock interface {
	Now() time.Time
}

//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodelrequests/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciliation triggered for authorization model request")
	reconcileTimestamp := r.Now()

	authorizationRequest := &extensionsv1.AuthorizationModelRequest{}
	if err := r.Get(ctx, req.NamespacedName, authorizationRequest); err != nil {
		logger.Error(err, "unable to fetch authorization model request", "authorizationModelRequestName", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	authorizationRequest.Status.State = extensionsv1.Synchronizing
	if err := r.Status().Update(ctx, authorizationRequest); err != nil {
		logger.Error(err, fmt.Sprintf("unable to set authorization model request in state %s", extensionsv1.Synchronizing), "authorizationModelRequestName", req.Name)
		r.Recorder.Event(
			authorizationRequest,
			v1.EventTypeWarning,
			string(EventReasonAuthorizationModelStatusChangeFailed),
			err.Error(),
		)
		return ctrl.Result{}, err
	}

	openFgaService, err := r.PermissionServiceFactory.GetService(r.Config)
	if err != nil {
		err = r.failAuthorizationModelRequestSynchronization(ctx, authorizationRequest, EventReasonClientInitializationFailed, err)
		logger.Error(err, "unable to get permission service")
		return ctrl.Result{}, err
	}

	err = r.ensureStoreExistsAndSetStoreId(ctx, req, openFgaService, authorizationRequest, &logger)
	if err != nil {
		err = r.failAuthorizationModelRequestSynchronization(ctx, authorizationRequest, EventReasonStoreFailed, err)
		logger.Error(err, "unable to get store")
		return ctrl.Result{}, err
	}

	authorizationModel, err := r.getAuthorizationModel(ctx, req, openFgaService, authorizationRequest, reconcileTimestamp, &logger)
	if err != nil {
		err = r.failAuthorizationModelRequestSynchronization(ctx, authorizationRequest, EventReasonAuthorizationModelCreationFailed, err)
		logger.Error(err, "unable to get authorization model")
		return ctrl.Result{}, err
	}

	if err = r.updateAuthorizationModel(ctx, openFgaService, authorizationRequest, authorizationModel, reconcileTimestamp, &logger); err != nil {
		err = r.failAuthorizationModelRequestSynchronization(ctx, authorizationRequest, EventReasonAuthorizationModelUpdateFailed, err)
		logger.Error(err, "unable to update authorization model")
		return ctrl.Result{}, err
	}

	authorizationRequest.Status.State = extensionsv1.Synchronized
	if err := r.Status().Update(ctx, authorizationRequest); err != nil {
		logger.Error(err, fmt.Sprintf("unable to set authorization model request in state %s", extensionsv1.Synchronized), "authorizationModelRequestName", req.Name)
		r.Recorder.Event(
			authorizationRequest,
			v1.EventTypeWarning,
			string(EventReasonAuthorizationModelStatusChangeFailed),
			err.Error(),
		)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AuthorizationModelRequestReconciler) failAuthorizationModelRequestSynchronization(ctx context.Context, authorizationRequest *extensionsv1.AuthorizationModelRequest, eventReason EventReason, err error) error {
	r.Recorder.Event(
		authorizationRequest,
		v1.EventTypeWarning,
		string(eventReason),
		err.Error(),
	)
	authorizationRequest.Status.State = extensionsv1.SynchronizationFailed
	if statusError := r.Status().Update(ctx, authorizationRequest); statusError != nil {
		return fmt.Errorf("failed to update status: %w with prior error %v", statusError, err)
	}
	return err
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

func (r *AuthorizationModelRequestReconciler) ensureStoreExistsAndSetStoreId(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) error {

	store := &extensionsv1.Store{}
	err := r.Get(ctx, req.NamespacedName, store)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return err
	case errors.IsNotFound(err):
		store, err = r.createStoreResource(ctx, req, openFgaService, authorizationModelRequest, log)
		if err != nil {
			return err
		}
	}
	openFgaService.SetStoreId(store.Spec.Id)
	return nil
}

func (r *AuthorizationModelRequestReconciler) createStoreResource(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfga.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.Store, error) {

	var store *openfga.Store
	var err error
	if authorizationModelRequest.Spec.ExistingStoreId != "" {
		store, err = openFgaService.CheckExistingStoresById(ctx, authorizationModelRequest.Spec.ExistingStoreId)
	} else {
		store, err = openFgaService.CheckExistingStoresByName(ctx, req.Name)
	}
	if err != nil {
		return nil, err
	}
	if store == nil && authorizationModelRequest.Spec.ExistingStoreId != "" {
		return nil, fmt.Errorf("store with id %s does not exist", authorizationModelRequest.Spec.ExistingStoreId)
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

	deletePredicate := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			if _, ok := e.Object.(*extensionsv1.AuthorizationModelRequest); ok {
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1.AuthorizationModelRequest{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithEventFilter(deletePredicate).
		Complete(r)
}
