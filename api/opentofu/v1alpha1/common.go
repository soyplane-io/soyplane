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

// Package v1alpha1 contains API Schema definitions for the opentofu v1alpha1 API group.
// +kubebuilder:object:generate=true
// +groupName=opentofu.soyplane.io
package v1alpha1

// JobMetadataSpec defines metadata fields for the generated Job.
type ObjectMetadata struct {
	// Labels are key-value pairs attached to the object.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations provide additional metadata for the object.
	Annotations map[string]string `json:"annotations,omitempty"`
	// GenerateName is used to auto-generate a unique object name with the specified prefix.
	GenerateName string `json:"generateName,omitempty"`
}
