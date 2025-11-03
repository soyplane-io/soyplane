# Repository Guidelines

## Basic Guidelines:
- Plan.md contains the state of the current task always read it for context.
- ALWAYS Keep Plan.md up to date at all times.

## Project Structure & Module Organization
Soyplane is a Go 1.23 Kubernetes operator. Manager and agent entrypoints sit in `cmd/manager` and `cmd/agent`. CRDs in `api/opentofu/v1alpha1` define `TofuModule`, `TofuExecution`, and `TofuStack`. Controllers live in `internal/controller/opentofu`, agent helpers in `internal/agent`, and defaults in `internal/settings`. Packaging and RBAC live under `config/`, bundles land in `dist/`, docs in `docs/`, and cached tools in `bin/`.

## Architecture Snapshot
The manager reconciles OpenTofu CRDs with shared helpers to keep desired state aligned and expose status and annotations for GitOps tools. When an execution is approved, the agent clones the module into `/tmp/tf-module`, injects secret-backed values, runs OpenTofu, and exports results through ConfigMaps or Secrets.

## CRD Overview
- `TofuModule` (`api/opentofu/v1alpha1/tofumodule_types.go`) captures reusable module blueprints: Git source/workdir/version, backend state config, provider references, default variables, and output destinations. Modules embed an execution template that stacks reuse and surface phase plus last plan/apply summaries in status.
- `TofuExecution` (`api/opentofu/v1alpha1/tofuexecution_types.go`) is an immutable plan/apply request. Spec locks the action, module reference, engine/version, and optional job template; status records lifecycle phase with an execution summary (revision, timestamps, job name).
- `TofuStack` (`api/opentofu/v1alpha1/tofustack_types.go`) links a module template to operational workflows. Today each stack references one module and issues executions for plan/apply/drift via its execution template, with room to evolve toward multi-module orchestration. Status mirrors module health, including the last execution name.
- `TofuProvider` (`api/opentofu/v1alpha1/tofuprovider_types.go`) stores shared provider blocks backed by ConfigMap/Secret refs. Modules reference providers by name; controllers/agents hydrate them at runtime.
- Shared helpers in `api/opentofu/v1alpha1/common.go` define reusable metadata and value-source structs, while `groupversion_info.go` registers all types with controller-runtime. See `docs/crds.md` for the full design narrative and interaction details.

## Build, Test, and Development Commands
- `make build` / `make build-agent` compile the binaries with manifest regeneration.
- `make run` / `make run-agent` run controller or agent locally; the agent preloads demo `TOFU_EXECUTION_*` vars.
- `make test` runs envtest-powered unit suites and writes `cover.out`; inspect via `go tool cover -html=cover.out`.
- `make test-e2e` drives Ginkgo smoke tests (Kind and Docker required).
- `make lint` / `make lint-fix` wrap `golangci-lint`.
- When controller-gen or other Go tooling fails with cache permission errors in sandboxed runs, create `./.cache/go-build` and invoke targets with `GOCACHE=$(pwd)/.cache/go-build make …`.
- Envtest needs to bind to loopback ports, so run `make test` with elevated permissions in the sandbox (or execute it directly on your host) to avoid bind errors.
- `make build-installer IMG=<registry>/soyplane:tag` regenerates `dist/install.yaml`; pair with `make docker-build` and `make deploy` to ship images.

## Coding Style & Naming Conventions
Format with `make fmt` and resolve lint feedback instead of suppressing it. Keep packages lowercase, exported identifiers PascalCase, locals camelCase, and ensure CRDs include Kubebuilder markers. YAML under `config/` follows the `<component>-kustomization.yaml` pattern and stays declarative; log via controller-runtime helpers rather than `fmt.Printf`.

## Testing Guidelines
Co-locate table-driven `TestXxx` files with their packages and use Gomega expectations. Grow coverage when adding reconciliation branches and clean up Kind fixtures in `test/e2e` Ginkgo suites. Run `make test` before PRs and note extra e2e prerequisites in `docs/`.

## Commit & Pull Request Guidelines
Use short, present-tense subjects (`adds opentofu sync`) under 72 characters, optional wrapped bodies, and reference issues with `Fixes #123`. PRs should describe scope, link tickets, list executed `make` targets, and share manifest diffs or logs for operator-visible changes.

## Security & Configuration Tips
Override images with `IMG` (`make build IMG=ghcr.io/soyplane/controller:dev`). Update RBAC rules in `config/rbac/` and rerun `make manifests` after permission changes. The agent reads `TOFU_EXECUTION_NAME`, `TOFU_EXECUTION_NAMESPACE`, and Git creds from secrets; document new env vars in `docs/` and keep the `/tmp/tf-module` workspace ephemeral or relocate it for production pods.
