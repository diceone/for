package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration loaded from config.yaml.
type Config struct {
	InventoryFile string `yaml:"inventory_file"`
	SSHUser       string `yaml:"ssh_user"`
	SSHKeyPath    string `yaml:"ssh_key_path"`
	SSHPassword   string `yaml:"ssh_password"`
	// SSHPort is the remote SSH port. Defaults to 22 if unset.
	SSHPort        int    `yaml:"ssh_port"`
	// JumpHost is an optional bastion/jump host (host:port).
	JumpHost       string `yaml:"jump_host"`
	// KnownHostsFile for SSH host key verification. Defaults to insecure if unset.
	KnownHostsFile string `yaml:"known_hosts_file"`
	// ServicesPath is the base directory for service task files. Defaults to "services".
	ServicesPath string `yaml:"services_path"`
	RunLocally   bool   `yaml:"run_locally"`
	// Forks is the number of parallel host connections. Defaults to 5.
	Forks    int    `yaml:"forks"`
	FailFast bool   `yaml:"fail_fast"`
	LogFile  string `yaml:"log_file"`
	// VaultPasswordFile is the path to a file containing the vault decryption password.
	VaultPasswordFile string `yaml:"vault_password_file"`
	// GatherFacts controls whether remote host facts are collected before running tasks.
	GatherFacts bool `yaml:"gather_facts"`
	// InventoryScript is the path to an executable that returns a dynamic JSON inventory.
	InventoryScript string `yaml:"inventory_script"`
}

func LoadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.SSHPort == 0 {
		cfg.SSHPort = 22
	}
	if cfg.ServicesPath == "" {
		cfg.ServicesPath = "services"
	}
	if cfg.Forks == 0 {
		cfg.Forks = 5
	}

	return &cfg, nil
}
