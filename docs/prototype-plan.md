# Soyplane Plan-Only Prototype

This guide walks through the minimal end-to-end slice for the Soyplane prototype: applying the CRDs, deploying the controller manager, and triggering a single `opentofu plan` from a `TofuStack`. The prototype deliberately skips advanced features such as variable injection, remote state backends, and output publishing.

## Prerequisites

- Kubernetes cluster (Kind is sufficient) and `kubectl` configured for the target cluster.
- `opentofu` (or `terraform`) installed on the worker nodes that execute Jobs. The default image runs the OpenTofu CLI bundled with the controller settings.
- Git access to the sample module repository (`https://github.com/soyplane-io/sample-tofu-modules.git`). If private, ensure the Job has credentials (e.g., via Secret + `envFrom`).
- Soyplane repository cloned locally with Go toolchain available to build controller binaries.

## Build & Deploy

```sh
# From the repository root
make prototype-plan IMG=ghcr.io/soyplane/controller:dev
```

The helper builds manifests, deploys the controller, and applies the plan-only module + stack samples. Adjust `IMG` if you publish the controller image elsewhere.

## Sample Manifests

The `config/samples/` directory now contains a minimal plan-only setup:

| File | Purpose |
| --- | --- |
| `opentofu_v1alpha1_tofumodule.yaml` | Defines a module that points at `sample-tofu-modules/null-module` with no variables, outputs, or backend configuration. |
| `opentofu_v1alpha1_tofustack.yaml` | Creates a stack that references the module and triggers plan executions using the module’s template. |

Apply them in order:

```sh
kubectl apply -f config/samples/opentofu_v1alpha1_tofumodule.yaml
kubectl apply -f config/samples/opentofu_v1alpha1_tofustack.yaml
```

The stack controller will create a `TofuExecution`, which in turn spawns a Kubernetes Job that clones the module and runs `opentofu plan`.

## Observability & Verification

1. **Check Executions**
   ```sh
   kubectl get tofuexecutions -n default
   ````
   Status should progress through `Pending → Running → Succeeded` once the Job completes.

2. **Inspect Jobs**
   ```sh
   kubectl get jobs -n default
   kubectl logs job/<job-name> -n default
   ```
   Logs show the inline shell script cloning the module and running `opentofu plan`.

3. **Status Summary**
   Controller reconciliation updates `status.executionSummary` with the Job name, timestamps, and a short summary (“Plan completed successfully” or “Plan failed”).

## Prototype Limitations

- **No variable injection**: Module/spec variables, ConfigMap/Secret value sources, and provider hydration are ignored in this slice.
- **No backend/state handling**: The plan runs with default local state. Remote backends will be wired once the agent pipeline lands.
- **Outputs not published**: `TofuModule.spec.outputs` is not acted upon; Jobs do not export values to Secrets/ConfigMaps yet.
- **Controller shell script**: The `TofuExecution` controller generates a shell command that runs directly inside the Job. A future milestone will replace this with the Go-based agent to handle richer workflows.

## Cleanup

```sh
make prototype-clean IMG=ghcr.io/soyplane/controller:dev
```

## Next Steps

- Implement the dedicated agent binary (variable injection, backend configuration, output publishing).
- Extend samples with templated Secrets/ConfigMaps for credentials and module variables.
- Add apply/drift execution paths and document approval flows once the prototype matures.
