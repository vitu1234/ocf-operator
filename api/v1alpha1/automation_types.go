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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type AutomationStatusFields string
type AutomationInputType string
type ActionType string

const (
	AutomationStatusCreating  AutomationStatusFields = "Creating"
	AutomationStatusRunning   AutomationStatusFields = "Running"
	AutomationStatusError     AutomationStatusFields = "Error"
	AutomationStatusCompleted AutomationStatusFields = "Completed"

	AutomationInputTypeNumeric AutomationInputType = "Numeric"
	AutomationInputTypeState   AutomationInputType = "State"
	AutomationInputTypeTime    AutomationInputType = "Time"

	ActionTypeMessage ActionType = "Message"
	ActionTypeEvent   ActionType = "Event"
)

// AutomationSpec defines the desired state of Automation
type AutomationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Automation. Edit automation_types.go to remove/update
	Trigger Trigger `json:"trigger"`
	// +kubebuilder:validation:MinItems=1
	Action []Action `json:"action"`
	// +kubebuilder:validation:MinItems=1
	PostAction []Action `json:"postaction,omitempty"`
}

// action structure part of the Spec
type Action struct {
	// +kubebuilder:validation:Enum=Message;Event
	Type ActionType `json:"type"`
	// +kubebuilder:validation:MinItems=1
	Data []Data `json:"data,omitempty"`
}

// trigger structure part of the Spec
type Trigger struct {
	// +kubebuilder:validation:Enum=Numeric;Time;State
	InputType   AutomationInputType `json:"inputtype"`   // Inputtype: The trigger expects numeric data; could be sensor readings
	OnDevice    string              `json:"ondevice"`    //ondevice: Indicates the device which will be sending the values
	ForResource string              `json:"forresource"` // Forresource: Indicates which resource on the device is exposed with that data

	To         string `json:"to,omitempty"`         //  To: indicates that a trigger is set if the state changes to a particular defined state
	From       string `json:"from,omitempty"`       //  From: indicates that a trigger is set if the state changes
	At         string `json:"at,omitempty"`         // specifies the time trigger an action
	TimeFormat string `json:"timeformat,omitempty"` // whether to use 12hr or 24hr time format
	Max        *int64 `json:"max,omitempty"`        // max&min: are available for the numeric type. Determines threshold for an automation
	Min        *int64 `json:"min,omitempty"`
	Period     Period `json:"period,omitempty"` // Period: If given, will trigger when the condition has been true for X time
}

type Period struct {
	Hours   *int64 `json:"hours,omitempty"`
	Minutes *int64 `json:"minutes,omitempty"`
	Seconds *int64 `json:"seconds,omitempty"`
}

type Data struct {
	TargetDevice   string `json:"targetdevice,omitempty"`
	TargetResource string `json:"targetresource,omitempty"`

	Value  *bool  `json:"value,omitempty"`
	Period Period `json:"period,omitempty"` // Period: how long to run the

	Message string `json:"message,omitempty"`
	Email   string `json:"email,omitempty"`
}

// AutomationStatus defines the observed state of Automation
type AutomationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status  AutomationStatusFields `json:"status,omitempty"`
	Message string                 `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`

// Automation is the Schema for the automations API
type Automation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutomationSpec   `json:"spec,omitempty"`
	Status AutomationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AutomationList contains a list of Automation
type AutomationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Automation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Automation{}, &AutomationList{})
}
