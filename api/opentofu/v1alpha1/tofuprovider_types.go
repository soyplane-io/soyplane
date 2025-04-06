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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TofuProviderSpec defines the desired state of TofuProvider.
type TofuProviderSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Type         string                          `json:"type"`
	ValueSources map[string]ValueSource          `json:"valueSources,omitempty"`
	RawConfig    string                          `json:"rawConfig,omitempty"`
	Config       map[string]apiextensionsv1.JSON `json:"config,omitempty"` // templated YAML
}

type ValueSource struct {
	SecretRef    *SecretKeyRef    `json:"secretRef,omitempty"`
	ConfigMapRef *ConfigMapKeyRef `json:"configMapRef,omitempty"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ConfigMapKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// TofuProviderStatus defines the observed state of TofuProvider.
type TofuProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
