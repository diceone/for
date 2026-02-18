# for

**for** is a command-line tool designed to automate the execution of shell commands and scripts across multiple remote servers, similar to Ansible, but with a focus on simplicity and use of Go.

## Features

- Execute commands and scripts defined in a configuration file.
- Run ad hoc commands and scripts on specified groups of hosts.
- Secure SSH connections for remote command execution.
- Support for grouping hosts in an inventory file (comments and blank lines are ignored).
- Option to run commands and scripts locally without SSH.
- Service-based execution similar to Ansible roles.
- Configurable SSH port and services base path.
- CLI flag `--local` takes precedence over the `run_locally` config-file setting.
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

#### Playbook via SSH

```bash
./for -config config.yaml -playbook playbook.yaml
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
# ssh_port defaults to 22 if omitted
ssh_port: 22
# services_path defaults to "services" if omitted
services_path: "services"
# run_locally can also be set here; the -local CLI flag takes precedence
run_locally: false
```

## Inventory File

`hosts.ini` follows a simple INI-style format. Blank lines and lines starting with `#` are ignored.

```ini
# Web servers
[webservers]
192.168.1.10
192.168.1.11

# Database servers
[dbservers]
192.168.1.20
192.168.1.21
```

## Playbook File

`playbook.yaml`:

```yaml
- name: Apply webserver service
  hosts: webservers
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

- name: Upgrade packages
  command: sudo apt-get upgrade -y
```

## Project Structure

```
cmd/for/main.go          # Entry point & CLI flag parsing
pkg/config/config.go     # Config loading (YAML)
pkg/inventory/inventory.go # INI-style inventory parser
pkg/ssh/ssh.go           # SSH client (command + script execution)
pkg/tasks/tasks.go       # Playbook / ad hoc task runner
pkg/utils/utils.go       # Helper utilities (e.g. IsScript)
services/                # Service task definitions
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
