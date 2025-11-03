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
	"maps"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
	settings "github.com/soyplane-io/soyplane/internal/settings"
)

// TofuExecutionReconciler reconciles a TofuExecution object
type TofuExecutionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofuexecutions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

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

	summaryChanged := false
	if execution.Status.ExecutionSummary.JobName != job.Name {
		execution.Status.ExecutionSummary.JobName = job.Name
		summaryChanged = true
	}

	if job.Status.StartTime != nil {
		if execution.Status.ExecutionSummary.StartedAt == nil ||
			!execution.Status.ExecutionSummary.StartedAt.Equal(job.Status.StartTime) {
			execution.Status.ExecutionSummary.StartedAt = job.Status.StartTime.DeepCopy()
			summaryChanged = true
		}
	}

	if job.Status.CompletionTime != nil {
		if execution.Status.ExecutionSummary.FinishedAt == nil ||
			!execution.Status.ExecutionSummary.FinishedAt.Equal(job.Status.CompletionTime) {
			execution.Status.ExecutionSummary.FinishedAt = job.Status.CompletionTime.DeepCopy()
			summaryChanged = true
		}
	}

	var summaryMsg string
	switch phase {
	case "Succeeded":
		summaryMsg = "Plan completed successfully"
	case "Failed":
		summaryMsg = "Plan failed"
	case "Running":
		summaryMsg = "Plan is running"
	default:
		summaryMsg = "Plan pending"
	}

	if execution.Status.ExecutionSummary.Summary != summaryMsg {
		execution.Status.ExecutionSummary.Summary = summaryMsg
		summaryChanged = true
	}

	statusChanged := summaryChanged
	if execution.Status.Phase != phase {
		execution.Status.Phase = phase
		statusChanged = true
	}

	if statusChanged {
		if err := r.Status().Update(ctx, &execution); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("TofuExecution status updated", "phase", phase, "job", job.Name)
	} else {
		log.Info("TofuExecution reconciled, nothing to update", "phase", phase, "job", job.Name)
	}

	if phase == "Running" || phase == "Pending" {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

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
	engineName := engine.Name
	if engineName == "" {
		engineName = "tofu"
	}

	moduleName := types.NamespacedName{
		Namespace: execution.Spec.ModuleRef.Namespace,
		Name:      execution.Spec.ModuleRef.Name,
	}
	module := opentofuv1alpha1.TofuModule{}
	if err := r.Get(ctx, moduleName, &module); err != nil {
		return nil, err
	}
	workdir := module.Spec.Workdir
	if workdir == "" {
		workdir = "."
	}
	execCfg, err := settings.Execution()
	if err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}
	image := execCfg.DefaultImage
	switch engineName {
	case "terraform":
		if engine.Version != "" {
			image = "hashicorp/terraform:" + engine.Version
		}
	case "tofu":
		if engine.Version != "" {
			image = "ghcr.io/opentofu/opentofu:" + engine.Version
		}
	default:
		if engine.Version != "" {
			image = fmt.Sprintf("%s:%s", engineName, engine.Version)
		}
	}
	cmd := fmt.Sprintf(`mkdir workspace;
	cd workspace;
	git clone %s .;
	cd %s;
	%s init;
	%s %s`, module.Spec.Source, workdir, engineName, engineName, execution.Spec.Action)
	// TODO: replace the inline shell script with the dedicated agent binary once the agent pipeline
	// is implemented (will handle variables, state backends, and richer status reporting).
	jobLabels := maps.Clone(execution.Spec.JobTemplate.Metadata.Labels)
	if jobLabels == nil {
		jobLabels = make(map[string]string)
	}

	jobAnnotations := maps.Clone(execution.Spec.JobTemplate.Metadata.Annotations)
	if jobAnnotations == nil {
		jobAnnotations = make(map[string]string)
	}

	podLabels := maps.Clone(execution.Spec.JobTemplate.Metadata.Labels)
	if podLabels == nil {
		podLabels = make(map[string]string)
	}

	podAnnotations := maps.Clone(execution.Spec.JobTemplate.Metadata.Annotations)
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}

	newJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobName + "-",
			Namespace:    execution.Namespace,
			Labels:       jobLabels,
			Annotations:  jobAnnotations,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
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
	if err := r.List(ctx, &childJobs, client.InNamespace(execution.Namespace)); err != nil {
		log.Error(err, "unable to list child job")
		return nil, err
	}
	ownedJobs := make([]*batchv1.Job, 0, len(childJobs.Items))
	for i := range childJobs.Items {
		job := &childJobs.Items[i]
		if metav1.IsControlledBy(job, execution) {
			ownedJobs = append(ownedJobs, job)
		}
	}
	switch len(ownedJobs) {
	case 0:
		return nil, nil
	case 1:
		return ownedJobs[0], nil
	default:
		// Sort newest first
		sort.SliceStable(ownedJobs, func(i, j int) bool {
			return ownedJobs[i].CreationTimestamp.Time.After(ownedJobs[j].CreationTimestamp.Time)
		})
		latest := ownedJobs[0]
		log.Info("Multiple Jobs owned by execution; keeping the newest and deleting stale ones", "execution", execution.Name, "activeJob", latest.Name, "staleCount", len(ownedJobs)-1)
		for _, stale := range ownedJobs[1:] {
			if err := r.Delete(ctx, stale); err != nil && !k8serrors.IsNotFound(err) {
				log.Error(err, "Failed to delete stale Job", "job", stale.Name)
			}
		}
		return latest, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TofuExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opentofuv1alpha1.TofuExecution{}).
		Owns(&batchv1.Job{}).
		Named("opentofu-tofuexecution").
		Complete(r)
}
