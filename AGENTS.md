# Repository Guidelines

## Project Structure & Module Organization
Soyplane is a Go 1.23 Kubernetes operator. Manager and agent entrypoints sit in `cmd/manager` and `cmd/agent`. CRDs in `api/opentofu/v1alpha1` define `TofuModule`, `TofuExecution`, and `TofuStack`. Controllers live in `internal/controller/opentofu`, agent helpers in `internal/agent`, and defaults in `internal/settings`. Packaging and RBAC live under `config/`, bundles land in `dist/`, docs in `docs/`, and cached tools in `bin/`.

## Architecture Snapshot
The manager reconciles OpenTofu CRDs with shared helpers to keep desired state aligned and expose status and annotations for GitOps tools. When an execution is approved, the agent clones the module into `/tmp/tf-module`, injects secret-backed values, runs OpenTofu, and exports results through ConfigMaps or Secrets.

## Build, Test, and Development Commands
- `make build` / `make build-agent` compile the binaries with manifest regeneration.
- `make run` / `make run-agent` run controller or agent locally; the agent preloads demo `TOFU_EXECUTION_*` vars.
- `make test` runs envtest-powered unit suites and writes `cover.out`; inspect via `go tool cover -html=cover.out`.
- `make test-e2e` drives Ginkgo smoke tests (Kind and Docker required).
- `make lint` / `make lint-fix` wrap `golangci-lint`.
- `make build-installer IMG=<registry>/soyplane:tag` regenerates `dist/install.yaml`; pair with `make docker-build` and `make deploy` to ship images.

## Coding Style & Naming Conventions
Format with `make fmt` and resolve lint feedback instead of suppressing it. Keep packages lowercase, exported identifiers PascalCase, locals camelCase, and ensure CRDs include Kubebuilder markers. YAML under `config/` follows the `<component>-kustomization.yaml` pattern and stays declarative; log via controller-runtime helpers rather than `fmt.Printf`.

## Testing Guidelines
Co-locate table-driven `TestXxx` files with their packages and use Gomega expectations. Grow coverage when adding reconciliation branches and clean up Kind fixtures in `test/e2e` Ginkgo suites. Run `make test` before PRs and note extra e2e prerequisites in `docs/`.

## Commit & Pull Request Guidelines
Use short, present-tense subjects (`adds opentofu sync`) under 72 characters, optional wrapped bodies, and reference issues with `Fixes #123`. PRs should describe scope, link tickets, list executed `make` targets, and share manifest diffs or logs for operator-visible changes.

## Security & Configuration Tips
Override images with `IMG` (`make build IMG=ghcr.io/soyplane/controller:dev`). Update RBAC rules in `config/rbac/` and rerun `make manifests` after permission changes. The agent reads `TOFU_EXECUTION_NAME`, `TOFU_EXECUTION_NAMESPACE`, and Git creds from secrets; document new env vars in `docs/` and keep the `/tmp/tf-module` workspace ephemeral or relocate it for production pods.
