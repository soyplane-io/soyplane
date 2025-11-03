# Milestone: Working Prototype (Controllers + Basic Plan)

## Goal
Deliver an end-to-end slice where the operator reconciles the OpenTofu CRDs and produces a successful `opentofu plan` through the agent. The prototype explicitly ignores variable injection, output publishing, and backend/state management—the module runs with its defaults only. Success criteria:
- apply minimal CRD manifests (`TofuModule`, `TofuStack`, `TofuExecution` as needed),
- observe controllers reconciling resources into Jobs,
- have the agent execute a plain OpenTofu plan (no variables, outputs, or remote state),
- watch status fields update to reflect success/failure.

## Task Breakdown

### Task 1 — Baseline Audit ✅ (Completed 2025-11-01)
1. Inspect `cmd/manager/main.go` to confirm controllers, webhooks, and CRDs are registered.
2. Review `config/rbac/` versus controller behaviour to ensure necessary permissions (Jobs, Pods, etc.).
3. Walk through `TofuModule`, `TofuExecution`, and `TofuStack` reconcilers for TODOs and blocking assumptions.
4. Cross-check CRD specs with agent expectations (repository URL, revision, workdir, execution template defaults).
5. Summarise findings in Plan.md before writing code.

**Findings**
- Manager wires all controllers, but `manager-role` lacks permissions for Jobs/Pods, so executions cannot create workloads (`cmd/manager/main.go:235`, `config/rbac/role.yaml:7`).
- `TofuModule` only creates the first execution and never refreshes it when the template/spec changes (`internal/controller/opentofu/tofumodule_controller.go:62`).
- `TofuExecution` scripts assume `spec.workdir` is set; otherwise the generated command attempts to `cd` without a path (`internal/controller/opentofu/tofuexecution_controller.go:156`).
- Execution status updates only the phase; summary fields remain empty (not a blocker but noted).
- `TofuStack` neither sets `ModuleRef` on generated executions nor watches child executions, so stack status stays stale (`internal/controller/opentofu/tofustack_controller.go:143`, `...:169`).
- Agent runner clones the module but never invokes `opentofu plan` or reports results (`internal/agent/runner.go:24`).

### Task 2 — Controller Flow Implementation ✅ (Completed 2025-11-02)
1. Expand RBAC so the manager can list/create/update Jobs, Pods, ConfigMaps, and Secrets required by executions. **(done — controller role now includes Jobs & Pods permissions)**
2. Update `TofuModule` reconciliation to trigger new executions when templates change and avoid duplicate jobs. **(done — module generation annotations drive refreshes when the last execution has finished)**
3. Harden `TofuExecution` reconciliation: enforce a single Job per execution, default workdir to `.`, translate Job status into execution phase/summary, and requeue while the Job runs. **(done — stale Jobs are deleted, workdir defaults to `.`, status captures job metadata, and running plans requeue automatically)**
4. Wire `TofuStack` to launch plan executions (using the module’s template or stack override) and copy execution phase/name into stack status. **(done — stack generation annotations trigger refreshes and stack watches executions)**
5. Add helper utilities for owner filtering/ordering to reduce code duplication. **(done — shared helpers for generation annotations and terminal detection added)**
6. Write unit/envtest coverage for the plan-only paths across these reconcilers. **(done — envtest verifies job lifecycle/status updates and a fake-client unit test exercises duplicate pruning logic).**

### Task 3 — Agent Execution Path (Deferred for POC)
1. Document that the prototype runs `opentofu plan` via the controller-generated shell without the Go agent.
2. Leave TODOs in the controller indicating the intent to swap in the agent once variable/backends/output handling is needed.
3. Track requirements for the future agent implementation (workspace prep, command invocation, status publishing) in the next milestone backlog.

### Task 4 — Documentation & Samples ✅ (Completed 2025-11-02)
1. Prepare sample manifests under `config/samples/` for the plan-only scenario (module, stack, optional execution trigger). **(done — module & stack samples point at the public null-module without extra variables)**
2. Write a quickstart guide (e.g., `docs/prototype.md`) describing how to build binaries, deploy CRDs/controllers, and trigger a plan; highlight prototype limitations. **(done — see `docs/prototype-plan.md`)**
3. Document prerequisites (OpenTofu binary, git auth requirements) and troubleshooting tips (log locations, expected events). **(done — captured in the quickstart)**
4. Add optional helper scripts/Makefile targets if they streamline the workflow. **(done — `make prototype-plan` / `make prototype-clean`)**

### Task 5 — Validation & Tooling (In Progress)
1. Run `make build` and resolve compilation issues introduced by tasks 2–4. **(done — `make` succeeds with `GOCACHE=$(pwd)/.cache/go-build`)**
2. Execute `make test` with the local Go cache workaround; ensure new tests cover controller logic. **(done — `make test` passes with envtest)**
3. Perform a smoke test (Kind/envtest/manual) applying the sample manifests and verifying a Job runs `opentofu plan`. **(blocked in sandbox — `kubectl` cannot reach the cluster due to denied TCP binds)**
4. Capture observed limitations and open questions, adding them to Plan.md for the next milestone. **(pending — document once smoke test runs on an unrestricted cluster)**

## Status
- Task 1: ✅ Completed.
- Task 2: ✅ Completed.
- Task 3: 🔁 Deferred (controller shell handles prototype plan; agent earmarked for later).
- Tasks 4–5: Pending.
