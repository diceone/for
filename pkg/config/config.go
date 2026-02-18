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
	// SSHPort is the remote SSH port. Defaults to 22 if unset.
	SSHPort      int    `yaml:"ssh_port"`
	// ServicesPath is the base directory for service task files. Defaults to "services".
	ServicesPath string `yaml:"services_path"`
	RunLocally   bool   `yaml:"run_locally"`
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

	return &cfg, nil
}
