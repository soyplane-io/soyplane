package agent

import (
	"context"

	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/go-logr/logr"
	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
)

type Runner struct {
	exec   *opentofuv1alpha1.TofuExecution
	module *opentofuv1alpha1.TofuModule
	logger logr.Logger
}

func NewRunner(exec *opentofuv1alpha1.TofuExecution, module *opentofuv1alpha1.TofuModule, logger logr.Logger) *Runner {
	return &Runner{exec: exec, module: module, logger: logger}
}

func (r *Runner) Execute(ctx context.Context) error {
	r.logger.Info("Starting Terraform execution")

	clonePath := "/tmp/tf-module" // or mount an emptyDir

	if err := cloneGitModule(r.module.Spec.Source, r.module.Spec.Version, clonePath); err != nil {
		return fmt.Errorf("module checkout failed: %w", err)
	}
	modulePath := path.Join(clonePath, r.module.Spec.Workdir)
	r.logger.Info("Module cloned", "path", modulePath)
	// TODO:
	// - Checkout module
	// - Init
	// - Plan
	// - Apply (if autoApply or approved)
	// - Parse outputs
	// - Call output exporter

	r.logger.Info("Terraform execution finished successfully")
	return nil
}

func cloneGitModule(source string, version string, workDir string) error {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", source, ".")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	if version != "" {
		cmd = exec.Command("git", "checkout", version)
		cmd.Dir = workDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git checkout %s failed: %w", version, err)
		}
	}

	return nil
}
