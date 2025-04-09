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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type ObjectRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// TofuExecutionSpec defines the desired state of TofuExecution.
type TofuExecutionSpec struct {
	Action      string                   `json:"action"` // plan | apply
	ModuleRef   ObjectRef                `json:"moduleRef"`
	JobTemplate *batchv1.JobTemplateSpec `json:"jobTemplate,omitempty"`
}

// TofuExecutionStatus defines the observed state of TofuExecution.
type TofuExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase       string             `json:"phase,omitempty"`
	StartedAt   *metav1.Time       `json:"startedAt,omitempty"`
	FinishedAt  *metav1.Time       `json:"finishedAt,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	TriggeredBy string             `json:"triggeredBy,omitempty"`
	JobName     string             `json:"jobName,omitempty"`
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TofuExecution is the Schema for the tofuexecutions API.
type TofuExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TofuExecutionSpec   `json:"spec,omitempty"`
	Status TofuExecutionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TofuExecutionList contains a list of TofuExecution.
type TofuExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TofuExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TofuExecution{}, &TofuExecutionList{})
}
