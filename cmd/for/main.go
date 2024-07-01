package main

import (
    "flag"
    "fmt"
    "os"
    "for/pkg/config"
    "for/pkg/inventory"
    "for/pkg/tasks"
    "strings"
)

func main() {
    configFile := flag.String("config", "", "Path to the configuration file")
    playbookFile := flag.String("playbook", "", "Path to the playbook file")
    showHelp := flag.Bool("help", false, "Show help message")
    adHocTask := flag.String("t", "", "Ad hoc task to run on specified group (e.g., 'command')")
    adHocGroup := flag.String("g", "", "Group to run ad hoc task on")
    runLocally := flag.Bool("local", false, "Run commands and scripts locally without SSH")

    flag.Parse()

    if *showHelp || (*configFile == "" && *adHocTask == "" && *playbookFile == "") {
        flag.Usage()
        os.Exit(1)
    }

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

    // Handle ad hoc tasks
    if *adHocTask != "" && *adHocGroup != "" {
        tasks.RunAdHocCommand(inv, *adHocGroup, *adHocTask, cfg.SSHUser, cfg.SSHKeyPath, *runLocally)
        os.Exit(0)
    } else if *adHocTask != "" && *adHocGroup == "" {
        fmt.Println("Error: Group must be specified with -g for ad hoc tasks")
        os.Exit(1)
    }

    // Load and run playbook
    if *playbookFile != "" {
        playbook, err := tasks.LoadTasks(*playbookFile)
        if err != nil {
            fmt.Printf("Error loading playbook: %v\n", err)
            os.Exit(1)
        }
        tasks.RunPlaybook(playbook, inv, cfg.SSHUser, cfg.SSHKeyPath, cfg.RunLocally)
        os.Exit(0)
    }

    fmt.Println("No tasks or commands specified")
    os.Exit(1)
}
