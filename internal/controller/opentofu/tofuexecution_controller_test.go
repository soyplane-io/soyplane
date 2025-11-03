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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
)

var _ = Describe("TofuExecution Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		tofuexecution := &opentofuv1alpha1.TofuExecution{}
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

			By("creating the custom resource for the Kind TofuExecution")
			err = k8sClient.Get(ctx, typeNamespacedName, tofuexecution)
			if err != nil && errors.IsNotFound(err) {
				resource := &opentofuv1alpha1.TofuExecution{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: opentofuv1alpha1.TofuExecutionSpec{
						Action: "plan",
						ModuleRef: opentofuv1alpha1.ObjectRef{
							Name:      moduleNamespacedName.Name,
							Namespace: moduleNamespacedName.Namespace,
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
			resource := &opentofuv1alpha1.TofuExecution{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TofuExecution")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			module := &opentofuv1alpha1.TofuModule{}
			err = k8sClient.Get(ctx, moduleNamespacedName, module)
			if err == nil {
				By("Cleaning up the referenced TofuModule")
				Expect(k8sClient.Delete(ctx, module)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
		})
		It("should create a Job and update status when the Job completes", func() {
			controllerReconciler := &TofuExecutionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("Reconciling to create the Job")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, typeNamespacedName, tofuexecution)).To(Succeed())

			var job batchv1.Job
			Eventually(func() bool {
				jobList := &batchv1.JobList{}
				if err := k8sClient.List(ctx, jobList); err != nil {
					return false
				}
				for i := range jobList.Items {
					candidate := jobList.Items[i]
					if metav1.IsControlledBy(&candidate, tofuexecution) {
						job = candidate
						return true
					}
				}
				return false
			}, 5*time.Second, 200*time.Millisecond).Should(BeTrue())
			Expect(job.Namespace).To(Equal(typeNamespacedName.Namespace))
			Expect(job.Spec.Template.Spec.Containers).NotTo(BeEmpty())

			By("Marking the Job as complete")
			job.Status.Succeeded = 1
			now := metav1.NewTime(time.Now())
			job.Status.StartTime = &now
			Expect(k8sClient.Status().Update(ctx, &job)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, tofuexecution)).To(Succeed())
			Expect(tofuexecution.Status.Phase).To(Equal("Succeeded"))
			Expect(tofuexecution.Status.ExecutionSummary.JobName).To(Equal(job.Name))
			Expect(tofuexecution.Status.ExecutionSummary.Summary).To(ContainSubstring("success"))
		})

	})
})

var _ = Describe("TofuExecution job helper", func() {
	It("keeps the newest job and deletes older ones", func() {
		scheme := runtime.NewScheme()
		Expect(opentofuv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(batchv1.AddToScheme(scheme)).To(Succeed())

		exec := &opentofuv1alpha1.TofuExecution{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "exec",
				Namespace: "default",
				UID:       types.UID("exec-uid"),
			},
		}
		exec.SetGroupVersionKind(opentofuv1alpha1.GroupVersion.WithKind("TofuExecution"))

		oldJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "old-job",
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(time.Unix(1, 0)),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(exec, opentofuv1alpha1.GroupVersion.WithKind("TofuExecution")),
				},
			},
		}
		newJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "new-job",
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(time.Unix(2, 0)),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(exec, opentofuv1alpha1.GroupVersion.WithKind("TofuExecution")),
				},
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(exec, oldJob, newJob).Build()
		reconciler := &TofuExecutionReconciler{Client: fakeClient, Scheme: scheme}

		ctx := context.Background()
		reqJob, err := reconciler.job(ctx, exec)
		Expect(err).NotTo(HaveOccurred())
		Expect(reqJob.Name).To(Equal("new-job"))

		remaining := &batchv1.JobList{}
		Expect(fakeClient.List(ctx, remaining)).To(Succeed())
		Expect(remaining.Items).To(HaveLen(1))
		Expect(remaining.Items[0].Name).To(Equal("new-job"))
	})
})
