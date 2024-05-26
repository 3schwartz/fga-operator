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
	openfga "github.com/openfga/go-sdk"
	fgaClient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"github.com/openfga/language/pkg/go/transformer"
	appsV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
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
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AuthorizationModelRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	requeueResult := ctrl.Result{RequeueAfter: 45 * time.Second}

	authorizationRequest := &extensionsv1.AuthorizationModelRequest{}
	if err := r.Get(ctx, req.NamespacedName, authorizationRequest); err != nil {
		log.Error(err, "unable to fetch authorization model request", "authorizationModelRequestName", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	openFgaRunClient, err := r.createOpenFgaClient()
	if err != nil {
		return requeueResult, err
	}
	store := &extensionsv1.Store{}
	err = r.Get(ctx, req.NamespacedName, store)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return requeueResult, err
	case errors.IsNotFound(err):
		store, err = r.createStore(ctx, req, openFgaRunClient, authorizationRequest, &log)
		if err != nil {
			return requeueResult, err
		}
	}
	openFgaRunClient.SetStoreId(store.Spec.Id)

	authorizationModel := &extensionsv1.AuthorizationModel{}
	err = r.Get(ctx, req.NamespacedName, authorizationModel)
	switch {
	case client.IgnoreNotFound(err) != nil:
		return requeueResult, err
	case errors.IsNotFound(err):
		_, err := r.createAuthorizationModel(ctx, req, openFgaRunClient, authorizationRequest, &log)
		if err != nil {
			return requeueResult, err
		}
	}
	if err = openFgaRunClient.SetAuthorizationModelId(authorizationModel.Spec.Instance.Id); err != nil {
		return requeueResult, err
	}

	if err = r.updateAuthorizationModel(ctx, openFgaRunClient, authorizationRequest, authorizationModel, &log); err != nil {
		return requeueResult, err
	}

	var deployments appsV1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingLabels{extensionsv1.OpenFgaStoreLabel: store.Name}); err != nil {
		log.Error(err, "unable to list deployments")
		return requeueResult, err
	}

	updates := r.updateStoreIdOnDeployments(deployments, store)

	r.updateAuthorizationModelIdOnDeployment(deployments, updates, authorizationModel, &log)

	for _, deployment := range updates {
		r.updateDeployment(ctx, &deployment, &log)
	}

	return requeueResult, nil
}

type DeploymentIdentifier struct {
	namespace string
	name      string
}

func (r *AuthorizationModelRequestReconciler) updateAuthorizationModelIdOnDeployment(
	deployments appsV1.DeploymentList,
	updates map[DeploymentIdentifier]appsV1.Deployment,
	authorizationModel *extensionsv1.AuthorizationModel,
	log *logr.Logger,
) {
	for _, deployment := range deployments.Items {
		authInstance, err := authorizationModel.GetVersionFromDeployment(deployment)
		if err != nil {
			log.Error(err, "unable to get auth instance from deployment", "deploymentName", deployment.Name)
			continue
		}
		deploymentIdentifier := DeploymentIdentifier{namespace: deployment.Namespace, name: deployment.Name}
		if updatedDeployment, ok := updates[deploymentIdentifier]; ok {
			deployment = updatedDeployment
		}
		if !updateDeploymentEnvVar(&deployment, extensionsv1.OpenFgaAuthModelIdEnv, authInstance.Id) {
			log.V(1).Info("deployment had correct auth id", "authInstance", authInstance)
			continue
		}

		deployment.Annotations[extensionsv1.OpenFgaAuthIdUpdatedAtAnnotation] = r.Now().UTC().Format(time.RFC3339)
		deployment.Annotations[extensionsv1.OpenFgaAuthModelVersionLabel] = authInstance.Version

		updates[deploymentIdentifier] = deployment
	}
}

func (r *AuthorizationModelRequestReconciler) updateStoreIdOnDeployments(
	deployments appsV1.DeploymentList,
	store *extensionsv1.Store,
) map[DeploymentIdentifier]appsV1.Deployment {
	updates := map[DeploymentIdentifier]appsV1.Deployment{}
	for _, deployment := range deployments.Items {
		if updateDeploymentEnvVar(&deployment, extensionsv1.OpenFgaStoreIdEnv, store.Spec.Id) {
			deployment.Annotations[extensionsv1.OpenFgaStoreIdUpdatedAtAnnotation] = r.Now().UTC().Format(time.RFC3339)

			updates[DeploymentIdentifier{namespace: deployment.Namespace, name: deployment.Name}] = deployment
		}
	}

	return updates
}

func updateDeploymentEnvVar(deployment *appsV1.Deployment, envVarName, envVarValue string) bool {
	updated := false
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		hasEnv := false
		for j := range container.Env {
			env := &container.Env[j]
			if env.Name != envVarName {
				continue
			}
			hasEnv = true
			if env.Value != envVarValue {
				updated = true
				env.Value = envVarValue
			}
			break
		}
		if hasEnv {
			continue
		}
		container.Env = append(container.Env, corev1.EnvVar{Name: envVarName, Value: envVarValue})
		updated = true
	}

	return updated
}

func (r *AuthorizationModelRequestReconciler) updateDeployment(
	ctx context.Context,
	deployment *appsV1.Deployment,
	log *logr.Logger,
) {
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "unable to update deployment", "deploymentName", deployment.Name)
	}
	log.V(0).Info("deployment updated", "deploymentName", deployment.Name)
}

func (r *AuthorizationModelRequestReconciler) updateAuthorizationModel(
	ctx context.Context,
	openFgaRunClient *fgaClient.OpenFgaClient,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	authorizationModel *extensionsv1.AuthorizationModel,
	log *logr.Logger) error {

	if authorizationModelRequest.Spec.AuthorizationModel == authorizationModel.Spec.AuthorizationModel {
		return nil
	}

	generatedJsonString, err := transformer.TransformDSLToJSON(authorizationModelRequest.Spec.AuthorizationModel)
	if err != nil {
		return err
	}
	var body fgaClient.ClientWriteAuthorizationModelRequest
	if err := json.Unmarshal([]byte(generatedJsonString), &body); err != nil {
	}
	data, err := openFgaRunClient.WriteAuthorizationModel(ctx).Body(body).Execute()
	if err != nil {
		return err
	}
	log.V(0).Info("Created new instance of authorization model in OpenFGA", "authorizationModelBody", body, "authorizationModelData", data)

	authorizationModel.Spec.LatestModels = append(authorizationModel.Spec.LatestModels, authorizationModel.Spec.Instance)
	authorizationModel.Spec.AuthorizationModel = authorizationModelRequest.Spec.AuthorizationModel
	authorizationModel.Spec.Instance = extensionsv1.AuthorizationModelInstance{
		Id:        data.AuthorizationModelId,
		Version:   body.SchemaVersion,
		CreatedAt: &metav1.Time{Time: r.Now()},
	}
	if err = r.Update(ctx, authorizationModel); err != nil {
		log.Error(err, "unable to update authorization model in Kubernetes", "authorizationModel", authorizationModel)
		return err
	}

	log.V(0).Info("Updated authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return nil
}

func (r *AuthorizationModelRequestReconciler) createAuthorizationModel(
	ctx context.Context,
	req ctrl.Request,
	openFgaRunClient *fgaClient.OpenFgaClient,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.AuthorizationModel, error) {

	generatedJsonString, err := transformer.TransformDSLToJSON(authorizationModelRequest.Spec.AuthorizationModel)
	if err != nil {
		return nil, err
	}
	var body fgaClient.ClientWriteAuthorizationModelRequest
	if err := json.Unmarshal([]byte(generatedJsonString), &body); err != nil {
	}
	data, err := openFgaRunClient.WriteAuthorizationModel(ctx).Body(body).Execute()
	if err != nil {
		return nil, err
	}
	log.V(0).Info("Created authorization model in OpenFGA", "authorizationModelBody", body, "authorizationModelData", data)

	authorizationModel := &extensionsv1.AuthorizationModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"authorization-model": req.Name,
			},
		},
		Spec: extensionsv1.AuthorizationModelSpec{
			Instance: extensionsv1.AuthorizationModelInstance{
				Id:        data.AuthorizationModelId,
				Version:   body.SchemaVersion,
				CreatedAt: &metav1.Time{Time: r.Now()},
			},
			AuthorizationModel: authorizationModelRequest.Spec.AuthorizationModel,
		},
	}
	if err := ctrl.SetControllerReference(authorizationModelRequest, authorizationModel, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, authorizationModel); client.IgnoreAlreadyExists(err) != nil {
		log.Error(err, fmt.Sprintf("Failed to authorization model %s", req.Name))
		return nil, err
	}
	log.V(0).Info("Created authorization model in Kubernetes", "authorizationModel", authorizationModel)

	return authorizationModel, nil
}

func checkExistingStores(
	ctx context.Context,
	openFgaRunClient *fgaClient.OpenFgaClient,
	storeName string,
	namespace string,
) (*extensionsv1.Store, error) {
	pageSize := openfga.PtrInt32(10)
	options := fgaClient.ClientListStoresOptions{
		PageSize: pageSize,
	}
	for {
		stores, err := openFgaRunClient.ListStores(ctx).Options(options).Execute()
		if err != nil {
			return nil, err
		}
		for _, oldStore := range stores.Stores {
			if oldStore.Name == storeName {
				return extensionsv1.NewStore(storeName, namespace, oldStore.Id, oldStore.CreatedAt), nil
			}
		}
		if stores.ContinuationToken == "" {
			break
		}
		options = fgaClient.ClientListStoresOptions{
			PageSize:          pageSize,
			ContinuationToken: openfga.PtrString(stores.ContinuationToken),
		}
	}
	return nil, nil
}

func (r *AuthorizationModelRequestReconciler) createStore(
	ctx context.Context,
	req ctrl.Request,
	openFgaRunClient *fgaClient.OpenFgaClient,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (*extensionsv1.Store, error) {

	existing, err := checkExistingStores(ctx, openFgaRunClient, req.Name, req.Namespace)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	body := fgaClient.ClientCreateStoreRequest{Name: req.Name}
	store, err := openFgaRunClient.CreateStore(ctx).Body(body).Execute()
	if err != nil {
		return nil, err
	}
	log.V(0).Info("Created store in OpenFGA", "storeOpenFGA", store)

	storeResource := extensionsv1.NewStore(req.Name, req.Namespace, store.Id, r.Now())
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

func (r *AuthorizationModelRequestReconciler) createOpenFgaClient() (*fgaClient.OpenFgaClient, error) {
	return fgaClient.NewSdkClient(&fgaClient.ClientConfiguration{
		ApiUrl: r.Config.ApiUrl,
		Credentials: &credentials.Credentials{
			Method: credentials.CredentialsMethodApiToken,
			Config: &credentials.Config{
				ApiToken: r.Config.ApiToken,
			},
		},
	})
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
