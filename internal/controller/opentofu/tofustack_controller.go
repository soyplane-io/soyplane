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

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// TofuStackReconciler reconciles a TofuStack object
type TofuStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofustacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofustacks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentofu.soyplane.io,resources=tofustacks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TofuStack object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *TofuStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	var stack opentofuv1alpha1.TofuStack

	if err := r.Get(ctx, req.NamespacedName, &stack); err != nil {
		log.Error(err, "Unable to fetch TofuStack")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	lastExecution, err := r.lastExecution(ctx, &stack)
	if err != nil {
		return ctrl.Result{}, err
	}

	if lastExecution == nil {
		execution, err := r.createExecution(ctx, &stack)
		if err != nil {
			return ctrl.Result{}, err
		}

		log.Info("TofuStack reconciled, Created TofuExecution for stack", "execution", execution.Name)
		return ctrl.Result{}, nil
	}
	if updated, err := r.updateStackStatus(ctx, &stack, lastExecution); err != nil {
		return ctrl.Result{}, err
	} else if updated {
		log.Info("TofuStack reconciled, Status updated")
		return ctrl.Result{}, nil
	}
	log.Info("TofuStack reconciled, nothing to do")
	return ctrl.Result{}, nil
}

func (r *TofuStackReconciler) updateStackStatus(ctx context.Context, stack *opentofuv1alpha1.TofuStack, execution *opentofuv1alpha1.TofuExecution) (bool, error) {
	log := logf.FromContext(ctx)
	stackChanged := false

	if stack.Status.LastExecutionName != execution.Name {
		stack.Status.LastExecutionName = execution.Name
		stackChanged = true
	}

	if stack.Status.Phase != execution.Status.Phase {
		stack.Status.Phase = execution.Status.Phase
		stackChanged = true
	}

	if stackChanged {
		stack.Status.ObservedGeneration = stack.GetGeneration()
		if err := r.Status().Update(ctx, stack); err != nil {
			if errors.IsConflict(err) {
				log.Info("Conflict on status update — will retry naturally")
				return false, nil
			}
			return false, err
		}
	}

	return stackChanged, nil
}

func (r *TofuStackReconciler) lastExecution(ctx context.Context, stack *opentofuv1alpha1.TofuStack) (*opentofuv1alpha1.TofuExecution, error) {
	log := logf.FromContext(ctx)
	var childExecutions opentofuv1alpha1.TofuExecutionList
	if err := r.List(ctx, &childExecutions, client.InNamespace(stack.Namespace), client.MatchingFields{executionOwnerKey: stack.Name}); err != nil {
		log.Error(err, "unable to list child executions")
		return nil, err
	}
	if len(childExecutions.Items) == 0 {
		return nil, nil
	}

	// Sort newest first
	sort.SliceStable(childExecutions.Items, func(i, j int) bool {
		return childExecutions.Items[i].CreationTimestamp.Time.After(childExecutions.Items[j].CreationTimestamp.Time)
	})

	return &childExecutions.Items[0], nil // Latest execution
}

func (r *TofuStackReconciler) createExecution(ctx context.Context, stack *opentofuv1alpha1.TofuStack) (*opentofuv1alpha1.TofuExecution, error) {
	exec := opentofuv1alpha1.TofuExecution{
		Spec: stack.Spec.ExecutionTemplate.Spec,
	}
	generateName := stack.Spec.ExecutionTemplate.Metadata.GenerateName
	if generateName == "" {
		generateName = fmt.Sprintf("%s-", stack.Name)
	}
	exec.GenerateName = generateName
	exec.Namespace = stack.Namespace
	exec.Labels = map[string]string{
		"opentofu.soyplane.io/stack": stack.Name,
	}

	if err := ctrl.SetControllerReference(stack, &exec, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, &exec); err != nil {
		return nil, err
	}

	return &exec, nil
}

var (
	executionOwnerKey = ".metadata.controller"
)

// SetupWithManager sets up the controller with the Manager.
func (r *TofuStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(),
		&opentofuv1alpha1.TofuExecution{},
		executionOwnerKey,
		func(rawObj client.Object) []string {
			exec := rawObj.(*opentofuv1alpha1.TofuExecution)
			owner := metav1.GetControllerOf(exec)
			if owner == nil {
				return nil
			}
			if owner.Kind != "TofuStack" {
				return nil
			}
			return []string{owner.Name}
		}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&opentofuv1alpha1.TofuStack{}).
		Owns(&opentofuv1alpha1.TofuExecution{}).
		Named("opentofu_tofustack").
		Complete(r)
}
