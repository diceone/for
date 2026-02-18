# for

**for** is a command-line tool designed to automate the execution of shell commands and scripts across multiple remote servers, similar to Ansible, but with a focus on simplicity and use of Go.

## Features

- Execute commands and scripts defined in a configuration file.
- Run ad hoc commands and scripts on specified groups of hosts.
- **Parallel host execution** – configurable `--forks` / `forks:` concurrency.
- **Dry-run mode** (`--dry-run`) – prints tasks without executing.
- **Tag filtering** (`--tags`, `--skip-tags`) on plays and tasks.
- **Template variables** in task commands via `{{ .varname }}` syntax.
- **Handlers** – tasks triggered via `notify:` run once per host after all tasks.
- **`copy` task type** – upload local files to remote hosts.
- **`ignore_errors`** per task + global `--fail-fast` / `fail_fast:` flag.
- **SSH known-hosts verification** via `known_hosts_file:` config option.
- **SSH password authentication** in addition to key auth.
- **SSH jump host / bastion** support via `jump_host:`.
- **Inventory host variables** (`192.168.1.10 ssh_port=2222 ansible_user=admin`).
- **Inventory group variables** (`[group:vars]` sections).
- Configurable SSH port and services base path.
- CLI `--local` flag and `run_locally:` config option (CLI takes precedence).
- Structured logging to file (`--log-file` / `log_file:`).
- Proper error propagation – non-zero exit codes on failures.

## Releases

Pre-built binaries for Linux, macOS and Windows (amd64 & arm64) are published automatically on every version tag via GitHub Actions.

Download the latest release from the [Releases page](https://github.com/diceone/for/releases) or create a new release by pushing a tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

A `checksums.txt` (SHA-256) is included with every release.

## Installation

### Prerequisites

- Go 1.20 or later

### Build from Source

```bash
git clone https://github.com/diceone/for.git
cd for
go mod tidy
go build -o for ./cmd/for
```

## Usage

### Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `./config.yaml` | Path to the configuration file |
| `-playbook` | | Path to the playbook file |
| `-t` | | Ad hoc task / command to run |
| `-g` | | Host group for ad hoc tasks |
| `-local` | `false` | Run locally without SSH (overrides `run_locally` in config) |
| `-dry-run` | `false` | Print tasks without executing them |
| `-fail-fast` | `false` | Abort on first failure |
| `-forks` | `0` | Parallel host connections (0 = use config default of 5) |
| `-tags` | | Comma-separated tags to run (e.g. `install,nginx`) |
| `-skip-tags` | | Comma-separated tags to skip |
| `-log-file` | | Append log output to this file in addition to stdout |
| `-version` | | Print version and exit |
| `-help` | | Show help message |

### Examples

#### Local ad hoc command (no config required)

```bash
./for -local -t "uptime"
```

#### Local playbook (no config required)

```bash
./for -local -playbook playbook.yaml
```

#### Dry-run (preview without executing)

```bash
./for -config config.yaml -playbook playbook.yaml -dry-run
```

#### Run only tagged tasks

```bash
./for -config config.yaml -playbook playbook.yaml -tags install,config
```

#### Skip certain tags

```bash
./for -config config.yaml -playbook playbook.yaml -skip-tags deploy
```

#### Playbook via SSH with 20 parallel connections

```bash
./for -config config.yaml -playbook playbook.yaml -forks 20
```

#### Playbook via SSH, abort on first failure

```bash
./for -config config.yaml -playbook playbook.yaml -fail-fast
```

#### Ad hoc command via SSH on a host group

```bash
./for -config config.yaml -g webservers -t "uptime"
```

#### Ad hoc script via SSH on a host group

```bash
./for -config config.yaml -g webservers -t "/path/to/script.sh"
```

## Configuration File

`config.yaml`:

```yaml
inventory_file: "hosts.ini"
ssh_user: "yourusername"
ssh_key_path: "/path/to/your/private/key"
# ssh_password: "optional-password-auth"
# ssh_port defaults to 22 if omitted
ssh_port: 22
# jump_host: "bastion.example.com:22"
# known_hosts_file: "/home/user/.ssh/known_hosts"
# services_path defaults to "services" if omitted
services_path: "services"
# run_locally can also be set here; the -local CLI flag takes precedence
run_locally: false
# forks: parallel host connections (default 5)
forks: 10
# fail_fast: abort playbook on first error
fail_fast: false
# log_file: optional path to write log output
# log_file: "/var/log/for.log"
```

## Inventory File

`hosts.ini` supports INI-style groups, per-host variables, and group variables. Blank lines and `#` comments are ignored.

```ini
# Web servers
[webservers]
192.168.1.10
192.168.1.11 ssh_port=2222 ansible_user=deploy

[webservers:vars]
env=production

# Database servers
[dbservers]
192.168.1.20
```

## Playbook File

`playbook.yaml` supports play-level vars, handlers, tags, and per-task options:

```yaml
- name: Apply webserver service
  hosts: webservers
  tags: [web]
  vars:
    version: "1.5.0"
  handlers:
    - name: restart nginx
      command: systemctl restart nginx
  services:
    - service: example_service

- name: Apply dbserver service
  hosts: dbservers
  services:
    - service: example_service
```

## Service File Structure

Services live under `<services_path>/<service_name>/tasks/main.yaml`.

`services/example_service/tasks/main.yaml`:

```yaml
- name: Update package index
  command: sudo apt-get update
  tags: [install]

- name: Upgrade packages
  command: sudo apt-get upgrade -y
  tags: [install]
  ignore_errors: true

- name: Deploy version {{ .version }}
  command: deploy.sh {{ .version }}
  tags: [deploy]
  notify: restart nginx

- name: Upload config
  copy:
    src: ./nginx.conf
    dest: /etc/nginx/nginx.conf
  tags: [config]
  notify: restart nginx
```

## Project Structure

```
cmd/for/main.go                  # Entry point & CLI flag parsing
pkg/config/config.go             # Config loading (YAML)
pkg/inventory/inventory.go       # INI-style inventory parser (host vars, group vars)
pkg/logger/logger.go             # Structured logging (slog) with optional file output
pkg/ssh/ssh.go                   # SSH client (command, script, file copy, jump host)
pkg/tasks/tasks.go               # Playbook / ad hoc runner (parallel, handlers, tags, templates)
pkg/utils/utils.go               # Helper utilities (IsScript)
services/                        # Service task definitions
.github/workflows/ci.yaml        # CI: test & vet on every push/PR
.github/workflows/release.yaml   # Release: cross-platform binaries on version tags
```

## Development

```bash
# Build
go build -o for ./cmd/for

# Vet
go vet ./...
```

### Running Tests

Currently, tests are not implemented. Contributions are welcome!

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
