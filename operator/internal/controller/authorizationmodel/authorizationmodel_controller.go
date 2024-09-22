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
	"context"
	extensionsv1 "fga-operator/api/v1"
	"fga-operator/internal/observability"
	"github.com/go-logr/logr"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AuthorizationModelReconciler reconciles a AuthorizationModel object
type AuthorizationModelReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Clock
	ReconciliationInterval *time.Duration
}

type Clock interface {
	Now() time.Time
}

const (
	EventRecorderLabel = "EventRecorderLabelAuthorizationModelReconciler"
	deploymentIndexKey = ".metadata.labels." + extensionsv1.OpenFgaStoreLabel
)

type EventReason string

const (
	EventReasonStoreNotFound                    EventReason = "StoreNotFound"
	EventReasonAuthorizationModelNotFound       EventReason = "AuthorizationModelNotFound"
	EventReasonAuthorizationModelIdUpdateFailed EventReason = "AuthorizationModelIdUpdateFailed"
	EventReasonFailedListingDeployments         EventReason = "FailedListingDeployments"
	EventReasonFailedUpdatingDeployment         EventReason = "FailedUpdatingDeployment"
)

//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.fga-operator,resources=authorizationmodels/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciliation triggered for authorization model request")
	reconcileTimestamp := r.Now()

	requeueResult := ctrl.Result{RequeueAfter: *r.ReconciliationInterval}

	store := &extensionsv1.Store{}
	if err := r.Get(ctx, req.NamespacedName, store); err != nil {
		logger.Error(err, "unable to fetch store", "storeName", req.Name)
		r.Recorder.Event(
			store,
			v1.EventTypeWarning,
			string(EventReasonStoreNotFound),
			err.Error(),
		)
		return ctrl.Result{}, err
	}

	authorizationModel := &extensionsv1.AuthorizationModel{}
	if err := r.Get(ctx, req.NamespacedName, authorizationModel); err != nil {
		r.Recorder.Event(
			authorizationModel,
			v1.EventTypeWarning,
			string(EventReasonAuthorizationModelNotFound),
			err.Error(),
		)
		logger.Error(err, "unable to fetch authorization modem", "authorizationModelName", req.Name)
		return ctrl.Result{}, err
	}

	var deployments appsV1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingFields{deploymentIndexKey: store.Name}); err != nil {
		r.Recorder.Event(
			authorizationModel,
			v1.EventTypeWarning,
			string(EventReasonAuthorizationModelIdUpdateFailed),
			err.Error(),
		)
		logger.Error(err, "unable to list deployments")
		return ctrl.Result{}, err
	}

	updates := updateStoreIdOnDeployments(deployments, store, reconcileTimestamp)

	updateFailures := updateAuthorizationModelIdOnDeployment(deployments, updates, authorizationModel, reconcileTimestamp, &logger)
	for _, updateError := range updateFailures {
		r.Recorder.Event(
			&updateError.deployment,
			v1.EventTypeWarning,
			string(EventReasonFailedListingDeployments),
			updateError.err.Error(),
		)
	}

	for _, deployment := range updates {
		r.updateDeployment(ctx, &deployment, req.Name, &logger)
	}

	return requeueResult, nil
}

func (r *AuthorizationModelReconciler) updateDeployment(
	ctx context.Context,
	deployment *appsV1.Deployment,
	modelName string,
	log *logr.Logger,

) {
	if err := r.Update(ctx, deployment); err != nil {
		r.Recorder.Event(
			deployment,
			v1.EventTypeWarning,
			string(EventReasonFailedUpdatingDeployment),
			err.Error(),
		)
		log.Error(err, "unable to update deployment", "deploymentName", deployment.Name)
		return
	}
	observability.RecordDeploymentUpdated(deployment.Name, modelName)
	log.V(0).Info("deployment updated", "deploymentName", deployment.Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
			if object, ok := e.Object.(*extensionsv1.AuthorizationModel); ok {
				observability.RecordK8AuthorizationModelEvent(observability.Deleted, object.Name)
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1.AuthorizationModel{}).
		WithEventFilter(deletePredicate).
		Complete(r)
}
