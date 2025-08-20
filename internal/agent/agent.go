package agent

import (
	"context"
	"fmt"
	"time"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(opentofuv1alpha1.AddToScheme(scheme))
}

func Run(name, namespace string) error {
	log := ctrl.Log.WithName("agent")
	cfg := ctrl.GetConfigOrDie()

	cl, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exec, err := FetchTofuExecution(ctx, cl, name, namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch execution: %w", err)
	}

	module, err := FetchTofuModule(ctx, cl, exec.Spec.ModuleRef.Name, exec.Spec.ModuleRef.Namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch execution: %w", err)
	}

	runner := NewRunner(exec, module, log)
	if err := runner.Execute(ctx); err != nil {
		log.Error(err, "execution failed")
		// optionally: patch failure
		return err
	}

	log.Info("Execution succeeded")
	// optionally: patch success
	return nil
}
