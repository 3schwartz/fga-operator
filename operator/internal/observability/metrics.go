package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Label keys used in Prometheus metrics for tracking events related to authorization models, stores, and deployments.
const (
	// LabelEvent represents the type of event (e.g., created, updated) for an authorization model.
	LabelEvent = "event"

	// LabelModel represents the name of the authorization model set in the authorization model request.
	LabelModel = "model"

	// LabelDeployment represents the name of a deployment.
	LabelDeployment = "deployment"

	// LabelLocation represent if the entity has been saved as a CRD in Kubernetes or in OpenFGA.
	LabelLocation = "location"
)

var (
	authorizationModelsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorization_model_events_total",
			Help: "Total number of authorization models created, updated or delete",
		},
		[]string{LabelLocation, LabelEvent, LabelModel},
	)

	storesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stores_total",
			Help: "Total number of stores created.",
		},
		[]string{LabelLocation, LabelModel},
	)

	deploymentUpdatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deployment_updated_total",
			Help: "Total number of deployments updated.",
		},
		[]string{LabelDeployment, LabelModel},
	)
)

type storeEvent string

const (
	kubernetes storeEvent = "kubernetes"
	openFGA    storeEvent = "open_fga"
)

type AuthorizationEvent string

const (
	Created AuthorizationEvent = "created"
	Updated AuthorizationEvent = "updated"
	Deleted AuthorizationEvent = "deleted"
)

func RecordDeploymentUpdated(deploymentName, modelName string) {
	deploymentUpdatedTotal.With(prometheus.Labels{LabelDeployment: deploymentName, LabelModel: modelName}).Inc()
}

func RecordK8StoreEvent(modelName string) {
	storesTotal.With(prometheus.Labels{LabelLocation: string(kubernetes), LabelModel: modelName}).Inc()
}

func RecordOpenFgaStoreEvent(modelName string) {
	storesTotal.With(prometheus.Labels{LabelLocation: string(openFGA), LabelModel: modelName}).Inc()
}

func RecordK8AuthorizationModelEvent(event AuthorizationEvent, modelName string) {
	authorizationModelsTotal.With(prometheus.Labels{LabelLocation: string(kubernetes), LabelEvent: string(event), LabelModel: modelName}).Inc()
}

func RecordOpenFgaAuthorizationModels(modelName string) {
	authorizationModelsTotal.With(prometheus.Labels{LabelLocation: string(openFGA), LabelEvent: string(Created), LabelModel: modelName}).Inc()
}

func InitializeCustomMetrics() {
	metrics.Registry.MustRegister(authorizationModelsTotal, storesTotal, deploymentUpdatedTotal)
}
