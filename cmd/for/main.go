package main

import (
	"flag"
	"fmt"
	"os"

	"for/pkg/config"
	"for/pkg/inventory"
	"for/pkg/tasks"
)

const defaultConfigPath = "./config.yaml"

func main() {
	configFile := flag.String("config", defaultConfigPath, "Path to the configuration file")
	playbookFile := flag.String("playbook", "", "Path to the playbook file")
	showHelp := flag.Bool("help", false, "Show help message")
	adHocTask := flag.String("t", "", "Ad hoc task to run (e.g., 'command')")
	adHocGroup := flag.String("g", "", "Group to run ad hoc task on")
	runLocalFlag := flag.Bool("local", false, "Run commands and scripts locally without SSH")

	flag.Parse()

	if *showHelp || (*adHocTask == "" && *playbookFile == "") {
		flag.Usage()
		os.Exit(1)
	}

	// Local execution – no config or inventory required.
	if *runLocalFlag {
		if *adHocTask != "" {
			if err := tasks.RunLocalAdHocCommand(*adHocTask); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}

		if *playbookFile != "" {
			playbook, err := tasks.LoadTasks(*playbookFile)
			if err != nil {
				fmt.Printf("Error loading playbook: %v\n", err)
				os.Exit(1)
			}
			if err := tasks.RunPlaybook(playbook, nil, "", "", 0, tasks.DefaultServicesPath, true); err != nil {
				fmt.Printf("Error running playbook: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// SSH / config-driven execution.
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	inv, err := inventory.LoadInventory(cfg.InventoryFile)
	if err != nil {
		fmt.Printf("Error loading inventory: %v\n", err)
		os.Exit(1)
	}

	// CLI flag takes precedence over the config-file setting.
	runLocally := *runLocalFlag || cfg.RunLocally

	if *adHocTask != "" {
		if *adHocGroup == "" {
			fmt.Println("Error: Group must be specified with -g for ad hoc tasks")
			os.Exit(1)
		}
		if err := tasks.RunAdHocCommand(inv, *adHocGroup, *adHocTask, cfg.SSHUser, cfg.SSHKeyPath, cfg.SSHPort, runLocally); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *playbookFile != "" {
		playbook, err := tasks.LoadTasks(*playbookFile)
		if err != nil {
			fmt.Printf("Error loading playbook: %v\n", err)
			os.Exit(1)
		}
		if err := tasks.RunPlaybook(playbook, inv, cfg.SSHUser, cfg.SSHKeyPath, cfg.SSHPort, cfg.ServicesPath, runLocally); err != nil {
			fmt.Printf("Error running playbook: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	fmt.Println("No tasks or commands specified")
	os.Exit(1)
}
