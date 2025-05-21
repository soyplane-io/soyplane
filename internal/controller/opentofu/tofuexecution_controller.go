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
	"fmt"
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
)

// TofuExecutionReconciler reconciles a TofuExecution object
type TofuExecutionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TofuExecution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *TofuExecutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	var execution opentofuv1alpha1.TofuExecution
	if err := r.Get(ctx, req.NamespacedName, &execution); err != nil {
		log.Error(err, "Unable to fetch TofuExecution")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	job, err := r.job(ctx, &execution)
	if err != nil {
		return ctrl.Result{}, err
	}

	if job == nil {
		newJob, err := r.constructJobFromExecution(ctx, &execution)
		if err != nil {
			log.Error(err, "Unable to construct Job from TofuExecution")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, newJob); err != nil {
			log.Error(err, "Unable to create Job for TofuExecution")
			return ctrl.Result{}, err
		}

		log.Info("Created new Job for TofuExecution", "Job", newJob.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	var phase string
	if job.Status.Failed > 0 {
		phase = "Failed"
	} else if job.Status.Succeeded > 0 {
		phase = "Succeeded"
	} else if job.Status.Active > 0 || job.Status.Terminating != nil || job.Status.UncountedTerminatedPods != nil {
		phase = "Running"
	} else {
		phase = "Pending"
	}
	if execution.Status.Phase == phase {
		log.Info("TofuExecution reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	execution.Status.Phase = phase
	if err := r.Status().Update(ctx, &execution); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("TofuExecution reconciled")

	return ctrl.Result{}, nil
}

func (r *TofuExecutionReconciler) getModule(ctx context.Context, module *opentofuv1alpha1.TofuModule, name types.NamespacedName) error {
	if err := r.Get(ctx, name, module); err != nil {
		return err
	}
	return nil
}

func (r *TofuExecutionReconciler) constructJobFromExecution(ctx context.Context, execution *opentofuv1alpha1.TofuExecution) (*batchv1.Job, error) {
	jobName := execution.Name

	if execution.Spec.JobTemplate.Metadata.GenerateName != "" {
		jobName = execution.Spec.JobTemplate.Metadata.GenerateName
	}
	engine := execution.Spec.Engine
	moduleName := types.NamespacedName{
		Namespace: execution.Spec.ModuleRef.Namespace,
		Name:      execution.Spec.ModuleRef.Name,
	}
	module := opentofuv1alpha1.TofuModule{}
	if err := r.Get(ctx, moduleName, &module); err != nil {
		return nil, err
	}
	var image string
	if engine.Name == "terraform" {
		image = "hashicorp/terraform:" + engine.Version
	} else {
		image = "ghcr.io/opentofu/opentofu:" + engine.Version
	}
	cmd := fmt.Sprintf(`mkdir workspace;
	cd workspace;
	git clone %s .;
	cd %s;
	%s init;
	%s %s`, module.Spec.Source, module.Spec.Workdir, engine.Name, engine.Name, execution.Spec.Action)
	newJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobName + "-",
			Namespace:    execution.Namespace,
			Labels:       execution.Spec.JobTemplate.Metadata.Labels,
			Annotations:  execution.Spec.JobTemplate.Metadata.Annotations,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      execution.Spec.JobTemplate.Metadata.Labels,
					Annotations: execution.Spec.JobTemplate.Metadata.Annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: execution.Spec.JobTemplate.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:    "simulate-tofu",
							Image:   image,
							Command: []string{"sh", "-exc", cmd},
							Env:     execution.Spec.JobTemplate.Env,
							EnvFrom: execution.Spec.JobTemplate.EnvFrom,
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(execution, newJob, r.Scheme); err != nil {
		return nil, err
	}

	return newJob, nil
}

func (r *TofuExecutionReconciler) job(ctx context.Context, execution *opentofuv1alpha1.TofuExecution) (*batchv1.Job, error) {
	log := logf.FromContext(ctx)
	var childJobs batchv1.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(execution.Namespace), client.MatchingFields{jobOwnerKey: execution.Name}); err != nil {
		log.Error(err, "unable to list child job")
		return nil, err
	}
	switch len(childJobs.Items) {
	case 0:
		return nil, nil
	case 1:
		return &childJobs.Items[0], nil
	default:
		log.Error(nil, "More than one Job owned by the same execution", "Execution", execution.Name)
		// Sort newest first
		sort.SliceStable(childJobs.Items, func(i, j int) bool {
			return childJobs.Items[i].CreationTimestamp.Time.After(childJobs.Items[j].CreationTimestamp.Time)
		})
		return &childJobs.Items[0], nil // Latest execution
	}
}

var (
	jobOwnerKey = ".metadata.controller"
)

// SetupWithManager sets up the controller with the Manager.
func (r *TofuExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&batchv1.Job{},
		jobOwnerKey,
		func(rawObj client.Object) []string {
			job := rawObj.(*batchv1.Job)
			owner := metav1.GetControllerOf(job)
			if owner == nil {
				return nil
			}
			if owner.Kind != "TofuExecution" {
				return nil
			}
			return []string{owner.Name}
		}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&opentofuv1alpha1.TofuExecution{}).
		Owns(&batchv1.Job{}).
		Named("opentofu-tofuexecution").
		Complete(r)
}
