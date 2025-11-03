# Soyplane CRD Design

This document captures the responsibilities, key fields, and interactions between Soyplane’s custom resources.

## Shared Concepts

All resources live in the `opentofu.soyplane.io/v1alpha1` API group. Helper structs such as `ObjectMetadata`, `ObjectRef`, and value-source references are defined in `api/opentofu/v1alpha1/common.go` and reused across CRDs.

## TofuModule

**API**: `api/opentofu/v1alpha1/tofumodule_types.go`  
**Purpose**: Declarative definition of an OpenTofu/Terraform module that can be reused across environments.

### Spec Highlights
- `source`, `version`, `workdir`: identify the source repository/commit and working directory.
- `backend`: backend type plus structured configuration or secret-backed value sources for state storage.
- `providers`, `variables`, `valueSources`: describe provider blocks and module inputs (defaults/documentation).
- `outputs`: configure where execution outputs should be written (Secrets or ConfigMaps).
- `executionTemplate`: seed metadata/spec used when stacks trigger executions from this module.
- `autoApply`, `driftDetection`: module-level toggles for automation defaults.

### Status
- Tracks phase, `observedGeneration`, summaries of last plan/apply runs, and `lastExecution` name.

### Interactions
- Referenced by `TofuExecution.spec.moduleRef` and `TofuStack.spec.moduleTemplate`.
- Outputs defined here become the source for downstream wiring (e.g., stack dependencies).

## TofuExecution

**API**: `api/opentofu/v1alpha1/tofuexecution_types.go`  
**Purpose**: Immutable record of a single plan or apply run for a module.

### Spec Highlights
- `action`: `plan` or `apply`.
- `moduleRef`: namespaced reference to the target `TofuModule`.
- `jobTemplate`: optional metadata/env overrides for the spawned Kubernetes Job.
- `engine`: which CLI to run (`tofu` or `terraform`) and the version tag.

### Status
- Embeds `ExecutionSummary` (revision, timestamps, triggeredBy, jobName) and exposes a lifecycle `phase` plus optional `conditions`.

### Interactions
- Created directly by users or controllers (e.g., `TofuStack` reconciler).
- Controllers read settings and module metadata to render Jobs that execute the run.
- Outputs are written according to the module’s `outputs` configuration.

## TofuStack

**API**: `api/opentofu/v1alpha1/tofustack_types.go`  
**Purpose**: Operational binding between a module definition and the executions required to manage its lifecycle.

### Spec Highlights
- `moduleTemplate`: reference to the module the stack manages (note: JSON tag currently `moduleTemplate` despite field name `ModuleRef`).
- `executionTemplate`: template for the `TofuExecution` objects this stack will generate.
- `autoApply`, `driftDetection`: stack-level toggles controlling automation cadence.

### Status
- Mirrors module status fields (phase, observedGeneration, lastPlan/apply summaries, lastExecution).

### Interactions
- Reconciler watches the referenced module and stack state to decide when to mint new `TofuExecution` objects (plan/apply/drift checks).
- Consumes outputs from previous executions (per module `outputs` config) to coordinate with downstream resources.

## TofuProvider

**API**: `api/opentofu/v1alpha1/tofuprovider_types.go`  
**Purpose**: Reusable provider declarations that modules can reference.

### Spec Highlights
- `type`: provider name (e.g., `aws`, `google`).
- `valueSources`: map provider variables to Secret/ConfigMap references.
- `rawConfig` / `config`: hold provider configuration as raw text or structured JSON.

### Status
- Currently empty placeholder for future reconciliation data.

### Interactions
- Modules reference providers by name; controllers resolve referenced secrets/configmaps during execution.

## Interaction Summary

1. **Modules define the blueprint** for infrastructure units, including default variables, backend configuration, and output destinations.
2. **Stacks bind a module** to an operational workflow. In the current design each stack references a single module template but can evolve toward multi-module orchestration. Stacks trigger execution objects using their template and monitor completion.
3. **Executions represent immutable runs** (plan or apply). They pull module metadata and settings, run via the agent, and publish status plus outputs.
4. **Providers supply shared configuration** and credentials consumed during module runs.

Controllers in `internal/controller/opentofu` orchestrate these resources: the stack controller instantiates executions, while module and provider controllers (future work) can validate definitions or propagate status. Executions drive the agent, which reads settings, fetches module source, runs OpenTofu/Terraform, and materializes outputs as directed.
