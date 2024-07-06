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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthorizationModelRequestSpec defines the desired state of AuthorizationModelRequest
type AuthorizationModelRequestSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	Instances []AuthorizationModelRequestInstance `json:"instances,omitempty"`
}

// AuthorizationModelRequestStatus defines the observed state of AuthorizationModelRequest
type AuthorizationModelRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AuthorizationModelRequest is the Schema for the authorizationmodelrequests API
type AuthorizationModelRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthorizationModelRequestSpec   `json:"spec,omitempty"`
	Status AuthorizationModelRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AuthorizationModelRequestList contains a list of AuthorizationModelRequest
type AuthorizationModelRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthorizationModelRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthorizationModelRequest{}, &AuthorizationModelRequestList{})
}

type ModelVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func ModelVersionFromString(version string) (ModelVersion, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return ModelVersion{}, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ModelVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ModelVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return ModelVersion{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return ModelVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func (v ModelVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

type AuthorizationModelRequestInstance struct {
	AuthorizationModel string       `json:"authorizationModel,omitempty"`
	Version            ModelVersion `json:"version,omitempty"`
}
