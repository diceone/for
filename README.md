# for

**for** is a command-line tool designed to automate the execution of shell commands and scripts across multiple remote servers, similar to Ansible, but with a focus on simplicity and use of Go.

## Features

- Execute commands and scripts defined in a configuration file.
- Run ad hoc commands and scripts on specified groups of hosts.
- Secure SSH connections for remote command execution.
- Support for grouping hosts in an inventory file.
- Option to run commands and scripts locally.
- Service-based execution similar to Ansible roles.

## Installation

### Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/diceone/for.git
   cd for
   ```

2. Initialize the Go module (if not already initialized):
   ```bash
   go mod init github.com/diceone/for
   ```

3. Download dependencies:
   ```bash
   go mod tidy
   ```

4. Build the binary:
   ```bash
   go build -o for ./cmd/for
   ```

### Prerequisites

- Go 1.16 or later

## Usage

### Command-Line Options

- `-config string`: Path to the configuration file (default: `./config.yaml`).
- `-playbook string`: Path to the playbook file.
- `-help`: Show help message.
- `-t string`: Ad hoc task to run (e.g., 'command').
- `-g string`: Group to run ad hoc task on.
- `-local`: Run commands and scripts locally without SSH.

### Examples

#### Running Locally Without a Configuration File

Run an ad hoc task locally without specifying a configuration file or group:

```bash
./for -t "uptime" -local
```

Run a local playbook:

```bash
./for -playbook playbook.yaml -local
```

#### Running with a Configuration and Playbook File

1. Create a configuration file `config.yaml`:

   ```yaml
   inventory_file: "hosts.ini"
   ssh_user: "yourusername"
   ssh_key_path: "/path/to/your/private/key"
   run_locally: false
   ```

2. Create a playbook file `playbook.yaml`:

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

3. Create an inventory file `hosts.ini`:

   ```ini
   [webservers]
   192.168.1.10
   192.168.1.11

   [dbservers]
   192.168.1.20
   192.168.1.21
   ```

4. Create a service file `services/example_service/tasks/main.yaml`:

   ```yaml
   - name: Update package index
     command: sudo apt-get update

   - name: Upgrade packages
     command: sudo apt-get upgrade -y
   ```

5. Run the tool with the configuration and playbook file:

   ```bash
   ./for -config config.yaml -playbook playbook.yaml
   ```

#### Running Ad Hoc Commands with SSH

Run an ad hoc task on the `webservers` group:

```bash
./for -config config.yaml -g webservers -t "uptime"
```

#### Running Ad Hoc Scripts with SSH

Run a script on the `webservers` group:

```bash
./for -config config.yaml -g webservers -t "/path/to/script.sh"
```

## Configuration File Structure

The configuration file should be in YAML format and include the path to the inventory file, SSH user details, SSH key path, and a flag to indicate if commands should be run locally.

Example `config.yaml`:

```yaml
inventory_file: "hosts.ini"
ssh_user: "yourusername"
ssh_key_path: "/path/to/your/private/key"
run_locally: false
```

## Playbook File Structure

The playbook file should be in YAML format and include the services to be applied to the specified hosts.

Example `playbook.yaml`:

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

Services should be organized in directories under `services/` with a `tasks/main.yaml` file defining the tasks.

Example `services/example_service/tasks/main.yaml`:

```yaml
- name: Update package index
  command: sudo apt-get update

- name: Upgrade packages
  command: sudo apt-get upgrade -y
```

## Development

### Prerequisites

- Go 1.20 or later

### Building the Project

1. Clone the repository:
   ```bash
   git clone https://github.com/diceone/for.git
   cd for
   ```

2. Build the binary:
   ```bash
   go build -o for ./cmd/for
   ```

### Running Tests

Currently, tests are not implemented. Contributions are welcome!

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
