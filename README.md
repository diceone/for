# for

**for** is a lightweight, Ansible-inspired CLI tool written in Go for automating command and script execution across multiple remote servers.

## Features

### Core
- Execute commands and scripts defined in playbook YAML files.
- Run ad hoc commands on specified host groups.
- **Parallel host execution** – configurable `--forks` / `forks:` concurrency.
- **Dry-run mode** (`--dry-run`) – prints tasks without executing.
- **Tag filtering** (`--tags`, `--skip-tags`) on plays and tasks.
- **Template variables** in task commands via `{{ .varname }}` syntax.
- **Handlers** – tasks triggered via `notify:` run once per host after all tasks.
- **`copy` task type** – upload local files to remote hosts.
- **`ignore_errors`** per task + global `--fail-fast` / `fail_fast:` flag.
- Structured logging to file (`--log-file` / `log_file:`).
- Proper error propagation – non-zero exit codes on failures.

### SSH
- **SSH known-hosts verification** via `known_hosts_file:`.
- **SSH password authentication** in addition to key auth.
- **SSH jump host / bastion** support via `jump_host:`.
- **SSH connection pooling** (multiplexing) – connections are reused across tasks.
- **Inventory host variables** (`192.168.1.10 ssh_port=2222 ansible_user=admin`).
- **Inventory group variables** (`[group:vars]` sections).

### Task Control (v1.2.0)
- **`when`** – conditional task execution (Go template expression).
- **`with_items`** – loop over a list; `{{ .item }}` available in command.
- **`timeout`** – per-task timeout (e.g. `timeout: 30s`).
- **`retries` + `delay`** – automatic retry with configurable pause.
- **`register`** – store task output in a variable for later tasks.
- **`changed_when`** – custom condition to mark a task as changed.
- **Role dependencies** via `meta/main.yaml` (`dependencies:` list).

### Observability (v1.2.0)
- **ANSI-coloured output** – auto-detected terminal; green/yellow/red status lines.
- **PLAY RECAP** – summary table per host (ok / changed / failed / skipped / ignored).
- **Facts gathering** (`--gather-facts`) – collects OS, arch, kernel, hostname, distro etc. as template variables.

### Security (v1.2.0)
- **Vault encryption** – AES-256-GCM encrypted values in config (`$FORVAULT;…`).
  Decrypt with `--vault-password-file`.

### Inventory (v1.2.0)
- **Dynamic inventory** (`--inventory-script`) – run any executable that returns JSON.

## Releases

Pre-built binaries for Linux, macOS and Windows (amd64 & arm64) are published automatically on every version tag via GitHub Actions.

Download the latest release from the [Releases page](https://github.com/diceone/for/releases).

A `checksums.txt` (SHA-256) is included with every release.

## Installation

### Prerequisites

- Go 1.21 or later

### Build from Source

```bash
git clone https://github.com/diceone/for.git
cd for
go mod tidy
go build -o for ./cmd/for
```

## Configuration

`config.yaml`:

```yaml
inventory_file: hosts.ini
ssh_user: ubuntu
ssh_key_path: ~/.ssh/id_ed25519
ssh_password: ""           # or $FORVAULT;… encrypted value
ssh_port: 22
jump_host: ""              # host:port of bastion
known_hosts_file: ~/.ssh/known_hosts
services_path: services
run_locally: false
forks: 10
fail_fast: false
log_file: ""
gather_facts: false
vault_password_file: ""    # path to plaintext password file
inventory_script: ""       # path to dynamic inventory executable
```

## Inventory

Static (`hosts.ini`):

```ini
[webservers]
192.168.1.10 ssh_port=2222 ansible_user=deploy
192.168.1.11

[webservers:vars]
app_env=production
```

Dynamic (`--inventory-script ./inventory.sh`):
The script must print JSON to stdout:

```json
{
  "webservers": {
    "hosts": ["192.168.1.10", "192.168.1.11"],
    "vars": {"app_env": "production"}
  }
}
```

## Playbooks

```yaml
- name: Deploy web application
  hosts: webservers
  vars:
    app_version: "1.4.2"
  services:
    - service: nginx
    - service: app
  handlers:
    - name: reload nginx
      command: systemctl reload nginx
```

## Service / Role Structure

```
services/
  nginx/
    meta/
      main.yaml        # dependencies:
    tasks/
      main.yaml        # task list
  app/
    tasks/
      main.yaml
```

### Task fields

```yaml
- name: Install package
  command: apt-get install -y nginx
  tags: [setup]
  when: "{{ .os }} == linux"
  with_items:
    - nginx
    - curl
  timeout: 60s
  retries: 3
  delay: 5s
  register: install_result
  changed_when: "installed"
  notify: reload nginx
  ignore_errors: false

- name: Upload config
  copy:
    src: files/nginx.conf
    dest: /etc/nginx/nginx.conf
```

## CLI Reference

```
Usage of for:
  -config string          Path to configuration file (default "./config.yaml")
  -playbook string        Path to playbook YAML
  -t string               Ad hoc command to run
  -g string               Host group for ad hoc command
  -local                  Run locally without SSH
  -dry-run                Print tasks without executing
  -fail-fast              Abort on first failure
  -forks int              Parallel connections (0 = config default)
  -tags string            Comma-separated tags to run
  -skip-tags string       Comma-separated tags to skip
  -log-file string        Append output to this file
  -gather-facts           Collect host facts before running tasks
  -vault-password-file    Path to vault password file
  -inventory-script       Path to dynamic inventory executable
  -version                Print version and exit
  -help                   Show usage
```

## Vault Usage

Encrypt a value:

```go
import "for/pkg/vault"
enc, _ := vault.Encrypt("my-secret-password", "my-vault-passphrase")
// enc = "$FORVAULT;..."
```

Place the encrypted value in `config.yaml`:

```yaml
ssh_password: "$FORVAULT;..."
```

Run with:

```bash
for -playbook playbook.yaml --vault-password-file ~/.vault_pass
```

## CI/CD

GitHub Actions workflows:
- **CI** (`.github/workflows/ci.yaml`): `go test -race ./...` + `go vet` + `go build` on every push/PR.
- **Release** (`.github/workflows/release.yaml`): cross-compile for 5 platforms on every `v*.*.*` tag.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for the full release history.

| Version | Date       | Highlights |
|---------|------------|------------|
| v1.2.0  | 2026-02-19 | Vault (AES-256-GCM), `when`/`with_items`, timeout, retry, `register`, `changed_when`, role deps, facts, ANSI PLAY RECAP, SSH pool, dynamic inventory |
| v1.1.0  | 2025-xx-xx | Parallel forks, known-hosts, password auth, jump host, inventory host/group vars, handlers, tag filtering, template vars, `copy` task, dry-run, fail-fast, CI |
| v1.0.0  | 2025-xx-xx | Initial release – playbooks, ad hoc commands, SSH, config |
