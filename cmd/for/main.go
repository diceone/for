package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"for/pkg/config"
	"for/pkg/inventory"
	"for/pkg/logger"
	"for/pkg/tasks"
)

const defaultConfigPath = "./config.yaml"

// version is set at build time via -ldflags="-X main.version=<tag>".
var version = "dev"

func main() {
	configFile   := flag.String("config", defaultConfigPath, "Path to the configuration file")
	playbookFile := flag.String("playbook", "", "Path to the playbook file")
	showHelp     := flag.Bool("help", false, "Show help message")
	showVersion  := flag.Bool("version", false, "Print version and exit")
	adHocTask    := flag.String("t", "", "Ad hoc task / command to run")
	adHocGroup   := flag.String("g", "", "Group to run ad hoc task on")
	runLocalFlag := flag.Bool("local", false, "Run locally without SSH (overrides run_locally in config)")
	dryRun       := flag.Bool("dry-run", false, "Print tasks without executing them")
	failFast     := flag.Bool("fail-fast", false, "Abort on first failure")
	forks        := flag.Int("forks", 0, "Parallel host connections (0 = use config default)")
	tagsArg      := flag.String("tags", "", "Comma-separated tags to run")
	skipTagsArg  := flag.String("skip-tags", "", "Comma-separated tags to skip")
	logFile      := flag.String("log-file", "", "Optional log file path (appended to stdout)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("for %s\n", version)
		os.Exit(0)
	}

	if *showHelp || (*adHocTask == "" && *playbookFile == "") {
		flag.Usage()
		os.Exit(1)
	}

	// Initialise logger (stdout + optional file).
	cleanup, err := logger.Init(*logFile)
	if err != nil {
		fmt.Printf("Error initialising logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	parseTags := func(s string) []string {
		if s == "" {
			return nil
		}
		parts := strings.Split(s, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}

	// Local execution – no config or inventory required.
	if *runLocalFlag {
		localOpts := tasks.RunOptions{
			RunLocally:   true,
			DryRun:       *dryRun,
			FailFast:     *failFast,
			Forks:        *forks,
			Tags:         parseTags(*tagsArg),
			SkipTags:     parseTags(*skipTagsArg),
			ServicesPath: tasks.DefaultServicesPath,
		}

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
			if err := tasks.RunPlaybook(playbook, nil, localOpts); err != nil {
				fmt.Printf("Error: %v\n", err)
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

	// Override log file from CLI if provided.
	if *logFile == "" && cfg.LogFile != "" {
		cleanup, err = logger.Init(cfg.LogFile)
		if err != nil {
			fmt.Printf("Error initialising logger: %v\n", err)
			os.Exit(1)
		}
	}

	inv, err := inventory.LoadInventory(cfg.InventoryFile)
	if err != nil {
		fmt.Printf("Error loading inventory: %v\n", err)
		os.Exit(1)
	}

	effectiveForks := cfg.Forks
	if *forks > 0 {
		effectiveForks = *forks
	}

	opts := tasks.RunOptions{
		SSHUser:        cfg.SSHUser,
		SSHKeyPath:     cfg.SSHKeyPath,
		SSHPassword:    cfg.SSHPassword,
		SSHPort:        cfg.SSHPort,
		JumpHost:       cfg.JumpHost,
		KnownHostsFile: cfg.KnownHostsFile,
		ServicesPath:   cfg.ServicesPath,
		RunLocally:     *runLocalFlag || cfg.RunLocally,
		DryRun:         *dryRun,
		FailFast:       *failFast || cfg.FailFast,
		Forks:          effectiveForks,
		Tags:           parseTags(*tagsArg),
		SkipTags:       parseTags(*skipTagsArg),
	}

	if *adHocTask != "" {
		if *adHocGroup == "" {
			fmt.Println("Error: Group must be specified with -g for ad hoc tasks")
			os.Exit(1)
		}
		if err := tasks.RunAdHocCommand(inv, *adHocGroup, *adHocTask, opts); err != nil {
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
		if err := tasks.RunPlaybook(playbook, inv, opts); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	fmt.Println("No tasks or commands specified")
	os.Exit(1)
}
