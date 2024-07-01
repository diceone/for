package config

import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
)

type Config struct {
    InventoryFile string `yaml:"inventory_file"`
    SSHUser       string `yaml:"ssh_user"`
    SSHKeyPath    string `yaml:"ssh_key_path"`
    RunLocally    bool   `yaml:"run_locally"`
}

func LoadConfig(file string) (*Config, error) {
    data, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, err
    }

    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    if err != nil {
        return nil, err
    }

    return &cfg, nil
}
