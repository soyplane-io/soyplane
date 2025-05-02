/*
Copyright 2025 Othmane El Warrak.

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

// ExecutionTemplateSpec defines the metadata and spec for TofuExecutions triggered by this stack.
type ExecutionTemplateSpec struct {
	// Metadata defines labels, annotations, and name generation for the execution.
	Metadata ObjectMetadata `json:"metadata,omitempty"`
	// Spec defines the TofuExecution runtime configuration.
	Spec TofuExecutionSpec `json:"spec"` // Specification of the execution.
}

// DriftDetectionSpec configures drift detection settings for a TofuStack.
type DriftDetectionSpec struct {
	Enabled  bool            `json:"enabled"`            // Enables or disables drift detection.
	Interval metav1.Duration `json:"interval,omitempty"` // Interval between drift checks (e.g., "30m").
}

// TofuStackSpec defines the desired state of a TofuStack.
type TofuStackSpec struct {
	ModuleRef         ObjectRef             `json:"moduleTemplate"`           // Reference to a TofuModule.
	ExecutionTemplate ExecutionTemplateSpec `json:"executionTemplate"`        // Template used to generate TofuExecutions.
	AutoApply         bool                  `json:"autoApply,omitempty"`      // If true, applies changes automatically when drift is detected.
	DriftDetection    *DriftDetectionSpec   `json:"driftDetection,omitempty"` // Optional drift detection configuration.
}

// TofuStackStatus defines the observed state of a TofuStack.
type TofuStackStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase              string             `json:"phase,omitempty"`              // High-level lifecycle phase
	ObservedGeneration int64              `json:"observedGeneration,omitempty"` // For avoiding stale updates
	LastPlan           *ExecutionSummary  `json:"lastPlan,omitempty"`           // Info from most recent plan
	LastApply          *ExecutionSummary  `json:"lastApply,omitempty"`          // Info from most recent apply
	Conditions         []metav1.Condition `json:"conditions,omitempty"`         // Standard K8s-style condition set
	LastExecutionName  string             `json:"lastExecution,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tfstack
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="LastExecution",type=string,JSONPath=`.status.lastExecution`

// TofuStack is the Schema for the tofustacks API.
type TofuStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TofuStackSpec   `json:"spec,omitempty"`
	Status TofuStackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TofuStackList contains a list of TofuStack.
type TofuStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TofuStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TofuStack{}, &TofuStackList{})
}
