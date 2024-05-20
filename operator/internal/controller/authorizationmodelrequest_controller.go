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
	fgaClient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	openfgaInternal.Config
	Clock
}

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=extensions.openfga-controller,resources=authorizationmodelrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AuthorizationModelRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	store := &extensionsv1.Store{}

	err := r.Get(ctx, req.NamespacedName, store)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return ctrl.Result{}, err
	case errors.IsNotFound(err):
		store, err = r.createStore(ctx, req, &log)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AuthorizationModelRequestReconciler) createStore(ctx context.Context, req ctrl.Request, log *logr.Logger) (*extensionsv1.Store, error) {
	currentFgaClient, err := fgaClient.NewSdkClient(&fgaClient.ClientConfiguration{
		ApiUrl: r.Config.ApiUrl,
		Credentials: &credentials.Credentials{
			Method: credentials.CredentialsMethodApiToken,
			Config: &credentials.Config{
				ApiToken: r.Config.ApiToken,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	body := fgaClient.ClientCreateStoreRequest{Name: req.Name}
	store, err := currentFgaClient.CreateStore(ctx).Body(body).Execute()
	if err != nil {
		return nil, err
	}

	storeResource := &extensionsv1.Store{
		Spec: extensionsv1.StoreSpec{
			Id: store.Id,
		},
		Status: extensionsv1.StoreStatus{
			CreatedAt: &metav1.Time{Time: r.Now()},
		},
	}
	if err := r.Create(ctx, storeResource); client.IgnoreAlreadyExists(err) != nil {
		log.Error(err, fmt.Sprintf("Failed to create store %s", req.Name))
		return nil, err
	}

	return storeResource, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuthorizationModelRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&extensionsv1.AuthorizationModelRequest{}).
		Owns(&extensionsv1.AuthorizationModel{}).
		Owns(&extensionsv1.Store{}).
		Complete(r)
}
