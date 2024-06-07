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

package v1

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"
)

const OpenFgaAuthModelIdEnv = "OPENFGA_AUTH_MODEL_ID"
const OpenFgaStoreIdEnv = "OPENFGA_STORE_ID"

const OpenFgaStoreLabel = "openfga-store"
const OpenFgaAuthModelVersionLabel = "openfga-auth-model-version"

const OpenFgaAuthIdUpdatedAtAnnotation = "openfga-auth-id-updated-at"
const OpenFgaStoreIdUpdatedAtAnnotation = "openfga-store-id-updated-at"

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthorizationModelSpec defines the desired state of AuthorizationModel
type AuthorizationModelSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	Instance           AuthorizationModelInstance   `json:"instance,omitempty"`
	AuthorizationModel string                       `json:"authorizationModel,omitempty"`
	LatestModels       []AuthorizationModelInstance `json:"latestModels,omitempty"`
}

// AuthorizationModelStatus defines the observed state of AuthorizationModel
type AuthorizationModelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AuthorizationModel is the Schema for the authorizationmodels API
type AuthorizationModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthorizationModelSpec   `json:"spec,omitempty"`
	Status AuthorizationModelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AuthorizationModelList contains a list of AuthorizationModel
type AuthorizationModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthorizationModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthorizationModel{}, &AuthorizationModelList{})
}

type AuthorizationModelInstance struct {
	Id        string       `json:"id,omitempty"`
	Version   string       `json:"version,omitempty"`
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}

type ByCreatedAtDesc []AuthorizationModelInstance

func (a ByCreatedAtDesc) Len() int           { return len(a) }
func (a ByCreatedAtDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCreatedAtDesc) Less(i, j int) bool { return a[i].CreatedAt.After(a[j].CreatedAt.Time) }

func SortAuthorizationModelInstancesByCreatedAtDesc(instances []AuthorizationModelInstance) {
	sort.Sort(ByCreatedAtDesc(instances))
}

func FilterBySchemaVersion(instances []AuthorizationModelInstance, version string) []AuthorizationModelInstance {
	var filtered []AuthorizationModelInstance
	for _, instance := range instances {
		if instance.Version == version {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func NewAuthorizationModel(name, namespace, authModelId, version, authModel string, now time.Time) AuthorizationModel {
	return AuthorizationModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"authorization-model": name,
			},
		},
		Spec: AuthorizationModelSpec{
			Instance: AuthorizationModelInstance{
				Id:        authModelId,
				Version:   version,
				CreatedAt: &metav1.Time{Time: now},
			},
			AuthorizationModel: authModel,
		},
	}
}

type AuthorizationModelInterface interface {
	GetVersionFromDeployment(deployment v1.Deployment) (AuthorizationModelInstance, error)
}

func (a *AuthorizationModel) GetVersionFromDeployment(deployment v1.Deployment) (AuthorizationModelInstance, error) {
	version, ok := deployment.Labels[OpenFgaAuthModelVersionLabel]
	if ok {
		instances := append(a.Spec.LatestModels, a.Spec.Instance)
		filtered := FilterBySchemaVersion(instances, version)
		if len(filtered) == 0 {
			return AuthorizationModelInstance{}, fmt.Errorf("neither current or any latest models match version %s", version)
		}
		SortAuthorizationModelInstancesByCreatedAtDesc(filtered)
		return filtered[0], nil
	}
	return a.Spec.Instance, nil
}
