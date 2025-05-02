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

package opentofu

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Children[T client.Object, L client.ObjectList](ctx context.Context, c client.Client, parent client.Object, list L) ([]T, error) {
	if err := c.List(ctx, list, client.InNamespace(parent.GetNamespace())); err != nil {
		return nil, err
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}

	var result []T
	for _, obj := range items {
		o := obj.(client.Object)
		if metav1.IsControlledBy(o, parent) {
			result = append(result, o.(T))
		}
	}
	return result, nil
}
