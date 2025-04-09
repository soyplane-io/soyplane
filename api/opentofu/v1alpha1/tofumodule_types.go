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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type BackendSpec struct {
	Type         string                   `json:"type"`
	RawConfig    string                   `json:"rawConfig,omitempty"`
	Config       map[string]apiextv1.JSON `json:"config,omitempty"`
	ValueSources map[string]ValueFrom     `json:"valueSources,omitempty"`
}

type TofuProviderRef struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
}

type KeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ValueFrom struct {
	SecretRef    *KeyRef `json:"secretRef,omitempty"`
	ConfigMapRef *KeyRef `json:"configMapRef,omitempty"`
}

type OutputSpec struct {
	From string         `json:"from"`
	To   []OutputTarget `json:"to"`
}

type OutputTarget struct {
	// +kubebuilder:validation:Enum=Secret;ConfigMap
	Kind string `json:"kind"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// TofuModuleSpec defines the desired state of TofuModule.
type TofuModuleSpec struct {
	Source       string                   `json:"source"`
	Version      string                   `json:"version,omitempty"`
	Backend      BackendSpec              `json:"backend"`
	Providers    []TofuProviderRef        `json:"providers,omitempty"`
	Variables    map[string]apiextv1.JSON `json:"variables,omitempty"`
	ValueSources map[string]ValueFrom     `json:"valueSources,omitempty"`
	Outputs      []OutputSpec             `json:"outputs,omitempty"`
}

// TofuModuleStatus defines the observed state of TofuModule.
type TofuModuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TofuModule is the Schema for the tofumodules API.
type TofuModule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TofuModuleSpec   `json:"spec,omitempty"`
	Status TofuModuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TofuModuleList contains a list of TofuModule.
type TofuModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TofuModule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TofuModule{}, &TofuModuleList{})
}
