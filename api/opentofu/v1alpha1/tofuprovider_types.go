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
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TofuProviderSpec defines the configuration for a Terraform/OpenTofu provider block.
type TofuProviderSpec struct {
	// Type is the name of the provider (e.g., "aws", "google").
	Type string `json:"type"`
	// ValueSources maps variables to secrets or config maps.
	ValueSources map[string]ValueSource `json:"valueSources,omitempty"`
	// RawConfig is a raw HCL/YAML block (stringified).
	RawConfig string `json:"rawConfig,omitempty"`
	// Config is a templated YAML configuration parsed into structured JSON.
	Config map[string]apiextv1.JSON `json:"config,omitempty"` // templated YAML
}

type ValueSource struct {
	// ValueSource represents where to pull a single value from (secret or config map).
	SecretRef    *SecretKeyRef    `json:"secretRef,omitempty"`
	ConfigMapRef *ConfigMapKeyRef `json:"configMapRef,omitempty"`
}

type SecretKeyRef struct {
	// SecretKeyRef identifies a key within a Kubernetes Secret.
	Name string `json:"name"`
	// Key is the specific key within the Secret.
	Key string `json:"key"`
}

type ConfigMapKeyRef struct {
	// ConfigMapKeyRef identifies a key within a ConfigMap.
	Name string `json:"name"`
	// Key is the specific key within the ConfigMap.
	Key string `json:"key"`
}

// TofuProviderStatus holds observed state (currently unused).
type TofuProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=tfprov
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TofuProvider is the Schema for the tofuproviders API.
type TofuProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TofuProviderSpec   `json:"spec,omitempty"`
	Status TofuProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TofuProviderList contains a list of TofuProvider.
type TofuProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TofuProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TofuProvider{}, &TofuProviderList{})
}
