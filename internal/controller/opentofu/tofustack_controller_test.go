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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
)

var _ = Describe("TofuStack Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		tofustack := &opentofuv1alpha1.TofuStack{}
		moduleNamespacedName := types.NamespacedName{
			Name:      "test-module",
			Namespace: "default",
		}

		BeforeEach(func() {
			By("ensuring the referenced TofuModule exists")
			module := &opentofuv1alpha1.TofuModule{}
			err := k8sClient.Get(ctx, moduleNamespacedName, module)
			if err != nil && errors.IsNotFound(err) {
				module = &opentofuv1alpha1.TofuModule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      moduleNamespacedName.Name,
						Namespace: moduleNamespacedName.Namespace,
					},
					Spec: opentofuv1alpha1.TofuModuleSpec{
						Source: "https://example.com/repo.git",
						ExecutionTemplate: opentofuv1alpha1.ExecutionTemplateSpec{
							Spec: opentofuv1alpha1.TofuExecutionSpec{
								Action: "plan",
								ModuleRef: opentofuv1alpha1.ObjectRef{
									Name:      moduleNamespacedName.Name,
									Namespace: moduleNamespacedName.Namespace,
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, module)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the custom resource for the Kind TofuStack")
			err = k8sClient.Get(ctx, typeNamespacedName, tofustack)
			if err != nil && errors.IsNotFound(err) {
				resource := &opentofuv1alpha1.TofuStack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: opentofuv1alpha1.TofuStackSpec{
						ExecutionTemplate: opentofuv1alpha1.ExecutionTemplateSpec{
							Spec: opentofuv1alpha1.TofuExecutionSpec{
								Action: "plan",
								ModuleRef: opentofuv1alpha1.ObjectRef{
									Name:      moduleNamespacedName.Name,
									Namespace: moduleNamespacedName.Namespace,
								},
							},
						},
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &opentofuv1alpha1.TofuStack{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TofuStack")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			module := &opentofuv1alpha1.TofuModule{}
			err = k8sClient.Get(ctx, moduleNamespacedName, module)
			if err == nil {
				By("Cleaning up the referenced TofuModule")
				Expect(k8sClient.Delete(ctx, module)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}

			var executions opentofuv1alpha1.TofuExecutionList
			err = k8sClient.List(ctx, &executions, client.InNamespace("default"))
			Expect(err).NotTo(HaveOccurred())
			for i := range executions.Items {
				exec := &executions.Items[i]
				if metav1.IsControlledBy(exec, resource) {
					Expect(k8sClient.Delete(ctx, exec)).To(Succeed())
				}
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TofuStackReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
