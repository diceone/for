# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.2.0] – 2026-02-19

### Added

#### Security
- **Vault encryption** (`pkg/vault`) – AES-256-GCM encryption for sensitive config values.
  Values prefixed with `$FORVAULT;` are automatically decrypted at startup when
  `--vault-password-file` (or `vault_password_file:` in config) is provided.
  - `vault.Encrypt(plaintext, password) (string, error)`
  - `vault.Decrypt(ciphertext, password) (string, error)`
  - `vault.DecryptMap(map, password) error` – decrypts all string values in a map
  - `vault.IsEncrypted(s string) bool`
  - `vault.LoadPassword(file string) (string, error)`

#### Facts Gathering
- **`pkg/facts`** – collects system information as template variables.
  - `GatherLocal()` – runtime.GOOS/GOARCH, uname, hostname, fqdn.
  - `GatherRemote(host, cfg)` – SSH-based collection of os, arch, kernel,
    hostname, fqdn, distro, distro_version, cpu_count, total_memory.
  - Enable via `--gather-facts` CLI flag or `gather_facts: true` in config.
  - Facts are available as `{{ .os }}`, `{{ .arch }}`, `{{ .hostname }}` etc. in task commands.

#### Output & Observability
- **`pkg/printer`** – structured, ANSI-coloured console output.
  - Auto-detects terminal; colours can be overridden via `printer.ColorsEnabled`.
  - `PlayHeader`, `TaskHeader`, `HandlerHeader`, `HostHeader` banners.
  - `OK` (green), `Changed` (yellow), `Failed` (red), `Ignored`, `Skipped`, `DryRun` result lines.
  - `RegisterNote` – shows which variable captured task output.
  - `Recap([]HostSummary)` – **PLAY RECAP** table at end of playbook run with
    per-host counts: ok / changed / failed / skipped / ignored.

#### Task Control
- **`when`** – conditional execution; Go template expression evaluated against current vars.
  Task is skipped when result is `false`, `0`, `no`, or empty string.
- **`with_items`** – loop over a YAML list; current element available as `{{ .item }}`.
- **`timeout`** – per-task duration string (e.g. `timeout: 30s`, `timeout: 2m`).
  Task is aborted with an error after the deadline.
- **`retries` + `delay`** – automatic retry on failure (e.g. `retries: 3`, `delay: 5s`).
- **`register`** – captures combined stdout/stderr of a task into a named variable
  accessible in subsequent tasks.
- **`changed_when`** – Go template expression; overrides the default changed detection.
  Evaluated with `{{ .output }}` available as the task's output.

#### Role Dependencies
- **`meta/main.yaml`** support – declare `dependencies:` list for a service/role.
  `LoadServiceTasksWithDeps()` resolves the full dependency graph (cycle-safe)
  and prepends dependency tasks before the service's own tasks.

#### SSH Multiplexing
- **`ssh.Pool`** – connection pooling type that caches `*ssh.Client` instances
  keyed by `user@host:port`.
  - `NewPool() *Pool`
  - `Pool.RunCommandOutput(host, command, cfg)`
  - `Pool.RunScript(host, scriptPath, cfg)`
  - `Pool.CopyFile(host, src, dest, cfg)`
  - `Pool.Close()`
  - `RunPlaybook` creates a pool automatically (one connection per host, reused across services/tasks).
- **`ssh.RunCommandOutput(host, command, cfg)`** – stateless helper returning combined output as string.

#### Dynamic Inventory
- **`pkg/inventory/dynamic`** – `LoadDynamic(script string) (*Inventory, error)`.
  Executes an arbitrary script and parses its JSON stdout:
  ```json
  { "group": { "hosts": ["1.2.3.4"], "vars": {"key": "value"} } }
  ```
  Enabled via `--inventory-script` CLI flag or `inventory_script:` in config.

#### Config & CLI
- New config fields: `vault_password_file`, `gather_facts`, `inventory_script`.
- New CLI flags: `--vault-password-file`, `--gather-facts`, `--inventory-script`.
- `RunOptions.SSHPool` and `RunOptions.GatherFacts` fields.

### Changed
- `executeTask` now returns `(TaskResult, error)` instead of `error`.
- `runHostTasks` now returns `printer.HostSummary` instead of `bool`.
- All console output routed through `pkg/printer` (replaces bare `fmt.Printf` calls).
- `LoadServiceTasks` replaced by `LoadServiceTasksWithDeps` in `RunPlaybook` (backwards compatible).

### Fixed
- SSH client connections are now properly reused across tasks (pool), reducing
  connection overhead for large playbooks.

---

## [v1.1.0]

### Added
- **Parallel host execution** – `forks:` config key and `--forks` CLI flag;
  goroutines bounded by a semaphore channel.
- **SSH known-hosts verification** – `known_hosts_file:` config option using
  `golang.org/x/crypto/ssh/knownhosts`.
- **SSH password authentication** – `ssh_password:` config field.
- **SSH jump host / bastion** – `jump_host:` config field (host:port).
- **Inventory host variables** – inline `key=value` pairs on host lines.
- **Inventory group variables** – `[group:vars]` INI sections.
- **Handlers** – tasks triggered by `notify:` that run at most once per host
  after all regular tasks complete.
- **Tag filtering** – `tags:` on plays and tasks; `--tags` / `--skip-tags` flags.
- **Template variables** – Go `text/template` expansion in task `command:` fields
  using play `vars:`, group vars, and host vars.
- **`copy` task type** – `copy: {src:, dest:}` uploads a local file via SSH stdin pipe.
- **Dry-run mode** – `--dry-run` flag prints what would happen without executing.
- **`fail_fast` / `--fail-fast`** – abort playbook run on first host failure.
- **Log file** – `log_file:` config key and `--log-file` flag; uses
  `pkg/logger` (`slog`-based MultiWriter).
- **`pkg/logger`** – structured `slog` logger with optional file output.
- **14 unit tests** (inventory, tasks, utils packages) passing with `-race`.
- **CI workflow** – `.github/workflows/ci.yaml` runs `go test -race ./...`,
  `go vet`, and `go build` on every push/PR to `main`.

---

## [v1.0.0]

### Added
- Initial release.
- Playbook execution: YAML-defined plays with per-host SSH command execution.
- Ad hoc command / script execution against inventory groups.
- `config.yaml`-driven configuration (inventory, SSH user, key, port, services path).
- Static INI inventory (`hosts.ini`).
- Services directory structure (`services/<name>/tasks/main.yaml`).
- Local execution mode (`run_locally: true` / `--local`).
- `--version` flag (injected at build time via `-ldflags`).
- **GitHub Actions release workflow** – cross-compiles to 5 platforms
  (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) on
  every `v*.*.*` tag; publishes binaries + `checksums.txt` as GitHub Release assets.

### Security
- Migrated from `gopkg.in/yaml.v2` (CVEs) to `gopkg.in/yaml.v3`.
- Updated `golang.org/x/crypto` to v0.48.0 and `golang.org/x/sys` to v0.41.0
  to resolve Dependabot-reported vulnerabilities.

[v1.2.0]: https://github.com/diceone/for/releases/tag/v1.2.0
[v1.1.0]: https://github.com/diceone/for/releases/tag/v1.1.0
[v1.0.0]: https://github.com/diceone/for/releases/tag/v1.0.0
