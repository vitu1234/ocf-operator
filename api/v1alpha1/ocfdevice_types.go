/*
Copyright 2023.

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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OCFDeviceCurrentStatus string

const (
	OCFDeviceCreating   OCFDeviceCurrentStatus = "Creating"
	OCFDeviceOnBoarding OCFDeviceCurrentStatus = "OnBoarding"
	OCFDeviceDiscovery  OCFDeviceCurrentStatus = "Discovery"
	OCFDeviceRunning    OCFDeviceCurrentStatus = "Running"
	OCFDeviceError      OCFDeviceCurrentStatus = "Error"
	OCFDeviceNotFound   OCFDeviceCurrentStatus = "NotFound"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OCFDeviceSpec defines the desired state of OCFDevice
type OCFDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of OCFDevice. Edit ocfdevice_types.go to remove/update
	Id      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Owned   bool   `json:"owned,omitempty"`
	OwnerID string `json:"ownerId,omitempty"`

	PreferedResources []PreferedResources `json:"preferredResources,omitempty"`
}

type PreferedResources struct {
	Name string `json:"name,omitempty"`
}

// OCFDeviceStatus defines the observed state of OCFDevice
type OCFDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Options []Options              `json:"options,omitempty"`
	Status  OCFDeviceCurrentStatus `json:"status,omitempty"`
	Message string                 `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// OCFDevice is the Schema for the ocfdevices API
type OCFDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCFDeviceSpec   `json:"spec,omitempty"`
	Status OCFDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=ocfdevice
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`
// +kubebuilder:subresource:status
// OCFDeviceList contains a list of OCFDevice
type OCFDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCFDevice `json:"items"`
}

type Options struct {
	CertIdentity     string        `json:"certIdentity,omitempty"`
	DiscoveryTimeout time.Duration `json:"discoveryTimeout,omitempty"`

	MfgCert       string `json:"mfgCert,omitempty"`
	MfgKey        string `json:"mfgKey,omitempty"`
	MfgTrustCA    string `json:"mfgTrustCA,omitempty"`
	MfgTrustCAKey string `json:"mfgTrustCAKey,omitempty"`

	IdentityCert              string `json:"identityCert,omitempty"`
	IdentityKey               string `json:"identityKey,omitempty"`
	IdentityIntermediateCA    string `json:"identityIntermediateCA,omitempty"`
	IdentityIntermediateCAKey string `json:"identityIntermediateCAKey,omitempty"`
	IdentityTrustCA           string `json:"identityTrustCA,omitempty"`
	IdentityTrustCAKey        string `json:"identityTrustCAKey,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OCFDevice{}, &OCFDeviceList{})
}
