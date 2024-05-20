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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthorizationModelSpec defines the desired state of AuthorizationModel
type AuthorizationModelSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	Instance AuthorizationModelInstance `json:"instance,omitempty"`
}

// AuthorizationModelStatus defines the observed state of AuthorizationModel
type AuthorizationModelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	LatestModels []AuthorizationModelInstance `json:"latestModels,omitempty"`
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
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}
