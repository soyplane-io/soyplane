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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectRef defines a reference to another namespaced resource.
type ObjectRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// JobTemplateSpec provides a template for generating a Kubernetes Job.
type JobTemplateSpec struct {
	// Metadata defines labels, annotations, and name generation for the job.
	Metadata ObjectMetadata `json:"metadata"`
	// Env specifies environment variables to inject into the container.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// ServiceAccountName is the name of the service account to run the job under.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type EngineSpec struct {
	// +kubebuilder:validation:Enum=opentofu;terraform
	// Engine type: "opentofu" (OpenTofu) or "terraform" (HashiCorp Terraform)
	// +kubebuilder:default=opentofu
	Name string `json:"name,omitempty"`

	// Optional: specific image tag or version to use, e.g. "v1.6.2"
	// +kubebuilder:default=latest
	Version string `json:"version,omitempty"`
}

// TofuExecutionSpec defines the desired execution of a Terraform-compatible module.
type TofuExecutionSpec struct {
	// +kubebuilder:validation:Enum=plan;apply
	// Action specifies the execution type: "plan" or "apply".
	Action string `json:"action"` // plan | apply
	// ModuleRef references the TofuModule to be executed.
	ModuleRef ObjectRef `json:"moduleRef"`
	// JobTemplate optionally overrides the default job used to run the execution.
	JobTemplate JobTemplateSpec `json:"jobTemplate,omitempty"`
	// Engine specifies the engine (OpenTofu, Terraform) and it's version.
	Engine EngineSpec `json:"engine,omitempty"`
}

// ExecutionSummary captures metadata about a specific execution of a module.
type ExecutionSummary struct {
	Revision    string       `json:"revision,omitempty"`
	StartedAt   *metav1.Time `json:"startedAt,omitempty"`
	FinishedAt  *metav1.Time `json:"finishedAt,omitempty"`
	Summary     string       `json:"summary,omitempty"`
	TriggeredBy string       `json:"triggeredBy,omitempty"`
	JobName     string       `json:"jobName,omitempty"`
}

// TofuExecutionStatus defines the observed state of a TofuExecution.
type TofuExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ExecutionSummary `json:",inline"`
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	// Phase represents the current lifecycle state of the execution.
	Phase string `json:"phase,omitempty"`
	// Conditions contains detailed condition objects for execution transitions.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tfexec
// +kubebuilder:printcolumn:name="Action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
