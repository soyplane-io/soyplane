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
	"strconv"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// TofuModuleReconciler reconciles a TofuModule object
type TofuModuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofumodules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofumodules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofumodules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TofuModule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *TofuModuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	var module opentofuv1alpha1.TofuModule

	if err := r.Get(ctx, req.NamespacedName, &module); err != nil {
		log.Error(err, "Unable to fetch TofuModule")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	lastExecution, err := r.lastExecution(ctx, &module)
	if err != nil {
		log.Error(err, "Unable to fetch last TofuExecution")
		return ctrl.Result{}, err
	}

	desiredGeneration := strconv.FormatInt(module.GetGeneration(), 10)

	if lastExecution == nil {
		newExecution, err := r.createExecution(ctx, &module)
		if err != nil {
			log.Error(err, "Unable to create new TofuExecution")
			return ctrl.Result{}, err
		}

		log.Info("TofuModule reconciled, Created TofuExecution", "execution", newExecution.Name)
		return ctrl.Result{}, nil
	}
	currentGeneration := ""
	if lastExecution.Annotations != nil {
		currentGeneration = lastExecution.Annotations[moduleGenerationAnnotation]
	}
	if currentGeneration != desiredGeneration && isExecutionTerminal(lastExecution.Status.Phase) {
		newExecution, err := r.createExecution(ctx, &module)
		if err != nil {
			log.Error(err, "Unable to create new TofuExecution for updated module spec")
			return ctrl.Result{}, err
		}
		log.Info("Module spec changed — triggered new TofuExecution", "execution", newExecution.Name, "generation", desiredGeneration)
		return ctrl.Result{}, nil
	}

	if updated, err := r.updateModuleStatus(ctx, &module, lastExecution); err != nil {
		log.Error(err, "Could not update TofuModule status")
		return ctrl.Result{}, err
	} else if updated {
		log.Info("TofuModule reconciled, Status updated")
		return ctrl.Result{}, nil
	}
	log.Info("TofuModule reconciled, nothing to do")
	return ctrl.Result{}, nil
}

func (r *TofuModuleReconciler) updateModuleStatus(ctx context.Context, module *opentofuv1alpha1.TofuModule, execution *opentofuv1alpha1.TofuExecution) (bool, error) {
	log := logf.FromContext(ctx)
	moduleChanged := false

	if module.Status.LastExecutionName != execution.Name {
		module.Status.LastExecutionName = execution.Name
		moduleChanged = true
	}

	if module.Status.Phase != execution.Status.Phase {
		module.Status.Phase = execution.Status.Phase
		moduleChanged = true
	}

	if moduleChanged {
		module.Status.ObservedGeneration = module.GetGeneration()
		if err := r.Status().Update(ctx, module); err != nil {
			if errors.IsConflict(err) {
				log.Info("Conflict on status update — will retry naturally")
				return false, nil
			}
			return false, err
		}
	}

	return moduleChanged, nil
}

func (r *TofuModuleReconciler) lastExecution(ctx context.Context, module *opentofuv1alpha1.TofuModule) (*opentofuv1alpha1.TofuExecution, error) {
	log := logf.FromContext(ctx)
	var childExecutions opentofuv1alpha1.TofuExecutionList
	if err := r.List(ctx, &childExecutions, client.InNamespace(module.Namespace)); err != nil {
		log.Error(err, "unable to list child executions")
		return nil, err
	}
	ownedExecutions := make([]*opentofuv1alpha1.TofuExecution, 0, len(childExecutions.Items))
	for i := range childExecutions.Items {
		exec := &childExecutions.Items[i]
		if metav1.IsControlledBy(exec, module) {
			ownedExecutions = append(ownedExecutions, exec)
		}
	}
	if len(ownedExecutions) == 0 {
		return nil, nil
	}

	// Sort newest first
	sort.SliceStable(ownedExecutions, func(i, j int) bool {
		return ownedExecutions[i].CreationTimestamp.Time.After(ownedExecutions[j].CreationTimestamp.Time)
	})

	return ownedExecutions[0], nil // Latest execution
}

func (r *TofuModuleReconciler) createExecution(ctx context.Context, module *opentofuv1alpha1.TofuModule) (*opentofuv1alpha1.TofuExecution, error) {
	exec := opentofuv1alpha1.TofuExecution{
		Spec: module.Spec.ExecutionTemplate.Spec,
	}
	generateName := module.Spec.ExecutionTemplate.Metadata.GenerateName
	if generateName == "" {
		generateName = fmt.Sprintf("%s-", module.Name)
	}
	exec.GenerateName = generateName
	exec.Namespace = module.Namespace

	exec.Labels = maps.Clone(module.Spec.ExecutionTemplate.Metadata.Labels)
	if exec.Labels == nil {
		exec.Labels = make(map[string]string)
	}

	exec.Annotations = maps.Clone(module.Spec.ExecutionTemplate.Metadata.Annotations)
	if exec.Annotations == nil {
		exec.Annotations = make(map[string]string)
	}
	// TODO: consider moving the parent generation marker into execution status once we
	// have a richer status contract and the reconciler owns status updates.
	exec.Annotations[moduleGenerationAnnotation] = strconv.FormatInt(module.GetGeneration(), 10)

	exec.Spec.ModuleRef.Name = module.Name
	exec.Spec.ModuleRef.Namespace = module.Namespace

	if err := ctrl.SetControllerReference(module, &exec, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, &exec); err != nil {
		return nil, err
	}

	return &exec, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TofuModuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		For(&opentofuv1alpha1.TofuModule{}).
		Owns(&opentofuv1alpha1.TofuExecution{}).
		Named("opentofu-tofumodule").
		Complete(r)
}
