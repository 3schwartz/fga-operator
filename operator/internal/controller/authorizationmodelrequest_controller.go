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
	"github.com/go-logr/logr"
	appsV1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	openfgaInternal "openfga-controller/internal/openfga"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	extensionsv1 "openfga-controller/api/v1"
)

// AuthorizationModelRequestReconciler reconciles a AuthorizationModelRequest object
type AuthorizationModelRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	openfgaInternal.PermissionServiceFactory
	openfgaInternal.Config
	Clock
}

type Clock interface {
	Now() time.Time
}

//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	reconcileTimestamp := r.Now()

	requeueResult := ctrl.Result{RequeueAfter: 45 * time.Second}

	authorizationRequest := &extensionsv1.AuthorizationModelRequest{}
	if err := r.Get(ctx, req.NamespacedName, authorizationRequest); err != nil {
		logger.Error(err, "unable to fetch authorization model request", "authorizationModelRequestName", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	openFgaService, err := r.PermissionServiceFactory.GetService(r.Config)
	if err != nil {
		return requeueResult, err
	}

	store, err := r.getStore(ctx, req, openFgaService, authorizationRequest, &logger)
	if err != nil {
		return requeueResult, err
	}

	authorizationModel, err := r.getAuthorizationModel(ctx, req, openFgaService, authorizationRequest, reconcileTimestamp, &logger)
	if err != nil {
		return requeueResult, err
	}

	if err = r.updateAuthorizationModel(ctx, openFgaService, authorizationRequest, authorizationModel, reconcileTimestamp, &logger); err != nil {
		return requeueResult, err
	}

	var deployments appsV1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingLabels{extensionsv1.OpenFgaStoreLabel: store.Name}); err != nil {
		logger.Error(err, "unable to list deployments")
		return requeueResult, err
	}

	updates := updateStoreIdOnDeployments(deployments, store, reconcileTimestamp)

	updateAuthorizationModelIdOnDeployment(deployments, updates, authorizationModel, reconcileTimestamp, &logger)

	for _, deployment := range updates {
		r.updateDeployment(ctx, &deployment, &logger)
	}

	return requeueResult, nil
}

func (r *AuthorizationModelRequestReconciler) updateDeployment(
	ctx context.Context,
	deployment *appsV1.Deployment,
	log *logr.Logger,
) {
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "unable to update deployment", "deploymentName", deployment.Name)
		return
	}
	log.V(0).Info("deployment updated", "deploymentName", deployment.Name)
}

func (r *AuthorizationModelRequestReconciler) updateAuthorizationModel(
	ctx context.Context,
	openFgaService openfgaInternal.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	authorizationModel *extensionsv1.AuthorizationModel,
	reconcileTimestamp time.Time,
	log *logr.Logger) error {

	if authorizationModelRequest.Spec.Version == authorizationModel.Spec.Instance.Version &&
		authorizationModelRequest.Spec.AuthorizationModel == authorizationModel.Spec.AuthorizationModel {
		return nil
	}

	authModelId, err := openFgaService.CreateAuthorizationModel(ctx, authorizationModelRequest, log)
	if err != nil {
		return err
	}

	authorizationModel.Spec.LatestModels = append(authorizationModel.Spec.LatestModels, authorizationModel.Spec.Instance)
	authorizationModel.Spec.AuthorizationModel = authorizationModelRequest.Spec.AuthorizationModel
	authorizationModel.Spec.Instance = extensionsv1.AuthorizationModelInstance{
		Id:        authModelId,
		Version:   authorizationModelRequest.Spec.Version,
		CreatedAt: &metav1.Time{Time: reconcileTimestamp},
	}
	if err = r.Update(ctx, authorizationModel); err != nil {
		log.Error(err, "unable to update authorization model in Kubernetes", "authorizationModel", authorizationModel)
		return err
	}

	log.V(0).Info("Updated authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return nil
}

func (r *AuthorizationModelRequestReconciler) getAuthorizationModel(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfgaInternal.PermissionService,
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
	if err = openFgaService.SetAuthorizationModelId(authorizationModel.Spec.Instance.Id); err != nil {
		return nil, err
	}
	return authorizationModel, nil
}

func (r *AuthorizationModelRequestReconciler) createAuthorizationModel(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfgaInternal.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	reconcileTimestamp time.Time,
	log *logr.Logger) (*extensionsv1.AuthorizationModel, error) {

	authModelId, err := openFgaService.CreateAuthorizationModel(ctx, authorizationModelRequest, log)
	if err != nil {
		return nil, err
	}

	authorizationModel := extensionsv1.NewAuthorizationModel(req.Name, req.Namespace, authModelId, authorizationModelRequest.Spec.Version, authorizationModelRequest.Spec.AuthorizationModel, reconcileTimestamp)

	if err := ctrl.SetControllerReference(authorizationModelRequest, &authorizationModel, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, &authorizationModel); err != nil {
		log.Error(err, fmt.Sprintf("Failed to authorization model %s", req.Name))
		return nil, err
	}
	log.V(0).Info("Created authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return &authorizationModel, nil
}

func (r *AuthorizationModelRequestReconciler) getStore(
	ctx context.Context,
	req ctrl.Request,
	openFgaService openfgaInternal.PermissionService,
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
	openFgaService openfgaInternal.PermissionService,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.Store, error) {

	store, err := openFgaService.CheckExistingStores(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if store == nil {
		store, err = openFgaService.CreateStore(ctx, req.Name, log)
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
	log.V(0).Info("Created store in Kubernetes", "storeKubernetes", storeResource)

	return storeResource, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationModelRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}

	// TODO: Create index on deployment

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1.AuthorizationModelRequest{}).
		Owns(&extensionsv1.AuthorizationModel{}).
		Owns(&extensionsv1.Store{}).
		Complete(r)
}