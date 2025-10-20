# Soyplane Settings Guide

## Configuration Sources
- The manager reads configuration from one or more YAML files passed via the `--configs` flag; when no paths are provided it defaults to `config.yaml` in the working directory.
- Each file is loaded through `koanf`, merged into a single tree, and hydrated into the strongly typed `SoyplaneSettings` struct. Defaults are applied with `go-defaults` and validated with `go-playground/validator`, so missing required fields fail fast during startup or reload.

## Reload Semantics
- File providers are staged and validated before they become the active configuration. Watch hooks are attached only after the initial load succeeds; if watch registration fails, the previous settings remain in place and the operator may retry init.
- Subsequent reloads are transactional: when a watched file changes, the operator loads the new snapshot into a temporary tree and swaps it in only if parsing and validation succeed. On failure, the previous snapshot stays active and a `soyplane_settings_reload_failures_total{stage="reload"}` counter increments.
- Watch setup errors increment the same counter with `stage="watch"`. These paths also emit Kubernetes warning events (`SettingsWatchFailed`, `SettingsReloadFailed`, etc.) when the manager runs in a pod with `POD_NAME`/`POD_NAMESPACE` set. Metrics remain the visibility mechanism during local command-line runs.

## Local Development Tips
- Ensure test fixtures and ad-hoc configs set required fields such as `test: true` and `execution.defaultImage` to satisfy validation during `go test`.
- When running the manager binary outside Kubernetes, you can simulate event emission by exporting dummy `POD_NAME` and `POD_NAMESPACE` values before invoking the binary. This mirrors the Downward API configuration used in-cluster.
