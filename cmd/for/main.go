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
    runLocally := flag.Bool("local", false, "Run commands and scripts locally without SSH")

    flag.Parse()

    if *showHelp || (*adHocTask == "" && *playbookFile == "") {
        flag.Usage()
        os.Exit(1)
    }

    if *runLocally {
        // Handle ad hoc task locally without requiring config or group
        if *adHocTask != "" {
            tasks.RunLocalAdHocCommand(*adHocTask)
            os.Exit(0)
        }

        // Handle local playbook execution
        if *playbookFile != "" {
            playbook, err := tasks.LoadTasks(*playbookFile)
            if err != nil {
                fmt.Printf("Error loading playbook: %v\n", err)
                os.Exit(1)
            }
            tasks.RunLocalPlaybook(playbook)
            os.Exit(0)
        }
    } else {
        // Load configuration
        cfg, err := config.LoadConfig(*configFile)
        if err != nil {
            fmt.Printf("Error loading config: %v\n", err)
            os.Exit(1)
        }

        // Load inventory
        inv, err := inventory.LoadInventory(cfg.InventoryFile)
        if err != nil {
            fmt.Printf("Error loading inventory: %v\n", err)
            os.Exit(1)
        }

        // Handle ad hoc tasks with SSH
        if *adHocTask != "" {
            if *adHocGroup == "" {
                fmt.Println("Error: Group must be specified with -g for ad hoc tasks")
                os.Exit(1)
            }
            tasks.RunAdHocCommand(inv, *adHocGroup, *adHocTask, cfg.SSHUser, cfg.SSHKeyPath, *runLocally)
            os.Exit(0)
        }

        // Load and run playbook with SSH
        if *playbookFile != "" {
            playbook, err := tasks.LoadTasks(*playbookFile)
            if err != nil {
                fmt.Printf("Error loading playbook: %v\n", err)
                os.Exit(1)
            }
            tasks.RunPlaybook(playbook, inv, cfg.SSHUser, cfg.SSHKeyPath, cfg.RunLocally)
            os.Exit(0)
        }
    }

    fmt.Println("No tasks or commands specified")
    os.Exit(1)
}
