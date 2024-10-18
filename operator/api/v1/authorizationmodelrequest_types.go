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

// AuthorizationModelRequestStatusState defines the state of the AuthorizationModelRequest.
// This enumeration represents the various stages of the lifecycle for an AuthorizationModelRequest.
type AuthorizationModelRequestStatusState string

const (
	// Pending indicates that the AuthorizationModelRequest has been created
	// but is not yet being processed.
	Pending AuthorizationModelRequestStatusState = "Pending"

	// Synchronizing indicates that the AuthorizationModelRequest is currently
	// being synchronized or actively processed.
	Synchronizing AuthorizationModelRequestStatusState = "Synchronizing"

	// Synchronized indicates that the request has been successfully processed
	// and is stable, ready for changes or further updates.
	Synchronized AuthorizationModelRequestStatusState = "Synchronized"

	// SynchronizationFailed indicates that the AuthorizationModelRequest
	// encountered an error during processing or synchronization.
	SynchronizationFailed AuthorizationModelRequestStatusState = "SynchronizationFailed"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuthorizationModelRequestSpec defines the desired state of AuthorizationModelRequest
type AuthorizationModelRequestSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// ExistingStoreId specifies the ID of an existing store in the system.
	// Only applicable when migrating from existing infrastructure where the operator was not previously used.
	ExistingStoreId string                              `json:"existingStoreId,omitempty"`
	Instances       []AuthorizationModelRequestInstance `json:"instances,omitempty"`
}

// AuthorizationModelRequestStatus defines the observed state of AuthorizationModelRequest.
// It captures the current status of the request, tracking its progress through
// different stages of its lifecycle.
type AuthorizationModelRequestStatus struct {
	// Specifies the current state of the AuthorizationModelRequest.
	// Valid values are:
	// - "Pending" (default): The request has been created but processing has not yet started;
	// - "Synchronizing": The request is actively being synchronized or processed;
	// - "Synchronized": The request has been successfully processed and is stable, ready for changes or further updates;
	// - "SynchronizationFailed": The request encountered an error during synchronization or processing.
	// Defaults to "Pending" when the request is created.
	// +kubebuilder:default="Pending"
	State AuthorizationModelRequestStatusState `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AuthorizationModelRequest is the Schema for the authorizationmodelrequests API
type AuthorizationModelRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthorizationModelRequestSpec `json:"spec,omitempty"`

	//+kubebuilder:default:status={"state": "Pending"}

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
	// ExistingAuthorizationModelId specifies the ID of an existing authorization model in the system.
	// Only applicable when migrating from existing infrastructure where the operator was not previously used.
	ExistingAuthorizationModelId string       `json:"existingAuthorizationModelId,omitempty"`
	AuthorizationModel           string       `json:"authorizationModel,omitempty"`
	Version                      ModelVersion `json:"version,omitempty"`
}
