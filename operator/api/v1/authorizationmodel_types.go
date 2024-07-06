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
	Instances []AuthorizationModelInstance `json:"instances,omitempty"`
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

type AuthorizationModelDefinition struct {
	Id                 string
	AuthorizationModel string
	Version            ModelVersion
}

func NewAuthorizationModelDefinition(id string, authorizationModel string, version ModelVersion) AuthorizationModelDefinition {
	return AuthorizationModelDefinition{
		Id:                 id,
		AuthorizationModel: authorizationModel,
		Version:            version,
	}
}

func (d AuthorizationModelDefinition) IntoInstance(now time.Time) AuthorizationModelInstance {
	return AuthorizationModelInstance{
		Id:                 d.Id,
		AuthorizationModel: d.AuthorizationModel,
		Version:            d.Version,
		CreatedAt:          &metav1.Time{Time: now},
	}
}

type AuthorizationModelInstance struct {
	Id                 string       `json:"id,omitempty"`
	AuthorizationModel string       `json:"authorizationModel,omitempty"`
	Version            ModelVersion `json:"version,omitempty"`
	CreatedAt          *metav1.Time `json:"createdAt,omitempty"`
}

type ByVersionAndCreatedAtDesc []AuthorizationModelInstance

func (a ByVersionAndCreatedAtDesc) Len() int      { return len(a) }
func (a ByVersionAndCreatedAtDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByVersionAndCreatedAtDesc) Less(i, j int) bool {
	if a[i].Version.Major != a[j].Version.Major {
		return a[i].Version.Major > a[j].Version.Major
	}
	if a[i].Version.Minor != a[j].Version.Minor {
		return a[i].Version.Minor > a[j].Version.Minor
	}
	if a[i].Version.Patch != a[j].Version.Patch {
		return a[i].Version.Patch > a[j].Version.Patch
	}
	return a[i].CreatedAt.After(a[j].CreatedAt.Time)
}

func SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(instances []AuthorizationModelInstance) {
	sort.Sort(ByVersionAndCreatedAtDesc(instances))
}

func FilterBySchemaVersion(instances []AuthorizationModelInstance, version ModelVersion) []AuthorizationModelInstance {
	var filtered []AuthorizationModelInstance
	for _, instance := range instances {
		if instance.Version == version {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func NewAuthorizationModel(name, namespace string, definitions []AuthorizationModelDefinition, now time.Time) AuthorizationModel {
	instances := make([]AuthorizationModelInstance, len(definitions))
	for i, definition := range definitions {
		instances[i] = definition.IntoInstance(now)
	}
	return AuthorizationModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"authorization-model": name,
			},
		},
		Spec: AuthorizationModelSpec{
			Instances: instances,
		},
	}
}

func (a *AuthorizationModel) GetVersionFromDeployment(deployment v1.Deployment) (AuthorizationModelInstance, error) {
	if len(a.Spec.Instances) == 0 {
		return AuthorizationModelInstance{}, fmt.Errorf("no authorization model exists")
	}
	version, ok := deployment.Labels[OpenFgaAuthModelVersionLabel]
	if ok {
		modelVersion, err := ModelVersionFromString(version)
		if err != nil {
			return AuthorizationModelInstance{}, err
		}
		filtered := FilterBySchemaVersion(a.Spec.Instances, modelVersion)
		if len(filtered) == 0 {
			return AuthorizationModelInstance{}, fmt.Errorf("neither current or any latest models match version %s", version)
		}
		SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(filtered)
		return filtered[0], nil
	}

	SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(a.Spec.Instances)
	return a.Spec.Instances[0], nil
}
