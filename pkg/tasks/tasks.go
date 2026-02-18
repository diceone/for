package tasks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"text/template"

	"for/pkg/inventory"
	"for/pkg/ssh"
	"for/pkg/utils"
	"gopkg.in/yaml.v3"
)

// DefaultServicesPath is the default base directory for service task files.
const DefaultServicesPath = "services"

type Playbook []Play

type Play struct {
	Name     string                 `yaml:"name"`
	Hosts    string                 `yaml:"hosts"`
	Services []Service              `yaml:"services"`
	Handlers []Handler              `yaml:"handlers"`
	Vars     map[string]interface{} `yaml:"vars"`
	Tags     []string               `yaml:"tags"`
}

type Service struct {
	ServiceName string `yaml:"service"`
}

// Handler is a task that runs only when notified by another task.
type Handler struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

// CopyTask describes a local→remote file copy.
type CopyTask struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

type Task struct {
	Name         string    `yaml:"name"`
	Command      string    `yaml:"command"`
	Copy         *CopyTask `yaml:"copy"`
	IgnoreErrors bool      `yaml:"ignore_errors"`
	Tags         []string  `yaml:"tags"`
	Notify       string    `yaml:"notify"`
}

// RunOptions consolidates all execution parameters so function signatures stay stable.
type RunOptions struct {
	SSHUser        string
	SSHKeyPath     string
	SSHPassword    string
	SSHPort        int
	JumpHost       string
	KnownHostsFile string
	ServicesPath   string
	RunLocally     bool
	DryRun         bool
	FailFast       bool
	Forks          int
	Tags           []string
	SkipTags       []string
}

func LoadTasks(file string) (Playbook, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var playbook Playbook
	if err = yaml.Unmarshal(data, &playbook); err != nil {
		return nil, err
	}

	return playbook, nil
}

// LoadServiceTasks loads the task list for a named service from servicesPath.
func LoadServiceTasks(servicesPath, serviceName string) ([]Task, error) {
	if servicesPath == "" {
		servicesPath = DefaultServicesPath
	}
	serviceFilePath := filepath.Join(servicesPath, serviceName, "tasks", "main.yaml")
	data, err := os.ReadFile(serviceFilePath)
	if err != nil {
		return nil, err
	}

	var serviceTasks []Task
	if err = yaml.Unmarshal(data, &serviceTasks); err != nil {
		return nil, err
	}

	return serviceTasks, nil
}

// ---------------------------------------------------------------------------
// Tag helpers
// ---------------------------------------------------------------------------

// matchesTags returns true when a task should run given tag filters.
// A task with no tags always runs unless it matches a skip-tag.
func matchesTags(taskTags, filterTags, skipTags []string) bool {
	for _, st := range skipTags {
		for _, tt := range taskTags {
			if st == tt {
				return false
			}
		}
	}
	if len(filterTags) == 0 {
		return true
	}
	for _, ft := range filterTags {
		for _, tt := range taskTags {
			if ft == tt {
				return true
			}
		}
	}
	// If filterTags are set but the task has no tags, skip it.
	return false
}

// ---------------------------------------------------------------------------
// Template variable expansion
// ---------------------------------------------------------------------------

func expandVars(s string, vars map[string]interface{}) (string, error) {
	if len(vars) == 0 || s == "" {
		return s, nil
	}
	tmpl, err := template.New("").Option("missingkey=zero").Parse(s)
	if err != nil {
		return s, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return s, err
	}
	return buf.String(), nil
}

// mergeVars merges multiple var maps; later maps win on key conflicts.
func mergeVars(maps ...map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func hostVarsToInterface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// SSH config builder
// ---------------------------------------------------------------------------

// sshConfigFor creates an ssh.Config for a host, applying per-host variable overrides.
func sshConfigFor(host inventory.Host, opts RunOptions) ssh.Config {
	cfg := ssh.Config{
		User:           opts.SSHUser,
		KeyPath:        opts.SSHKeyPath,
		Password:       opts.SSHPassword,
		Port:           opts.SSHPort,
		JumpHost:       opts.JumpHost,
		KnownHostsFile: opts.KnownHostsFile,
	}
	if v, ok := host.Vars["ansible_user"]; ok {
		cfg.User = v
	}
	if v, ok := host.Vars["ssh_user"]; ok {
		cfg.User = v
	}
	if v, ok := host.Vars["ansible_port"]; ok {
		var p int
		if _, err := fmt.Sscan(v, &p); err == nil {
			cfg.Port = p
		}
	}
	if v, ok := host.Vars["ssh_port"]; ok {
		var p int
		if _, err := fmt.Sscan(v, &p); err == nil {
			cfg.Port = p
		}
	}
	return cfg
}

// ---------------------------------------------------------------------------
// Core task executor
// ---------------------------------------------------------------------------

func executeTask(task Task, host inventory.Host, opts RunOptions, vars map[string]interface{}) error {
	// Expand template variables
	cmd, err := expandVars(task.Command, vars)
	if err != nil {
		return fmt.Errorf("template expand: %w", err)
	}

	if opts.DryRun {
		if task.Copy != nil {
			fmt.Printf("[dry-run] COPY %s -> %s:%s\n", task.Copy.Src, host.Address, task.Copy.Dest)
		} else {
			fmt.Printf("[dry-run] CMD  %s\n", cmd)
		}
		return nil
	}

	if task.Copy != nil {
		if opts.RunLocally {
			return copyLocal(task.Copy.Src, task.Copy.Dest)
		}
		return ssh.CopyFile(host.Address, task.Copy.Src, task.Copy.Dest, sshConfigFor(host, opts))
	}

	if opts.RunLocally {
		if utils.IsScript(cmd) {
			return runLocalScript(cmd)
		}
		return runLocalCommand(cmd)
	}

	sshCfg := sshConfigFor(host, opts)
	if utils.IsScript(cmd) {
		return ssh.RunScript(host.Address, cmd, sshCfg)
	}
	return ssh.RunCommand(host.Address, cmd, sshCfg)
}

// runHostTasks executes all service tasks for one host and fires notified handlers.
// Returns true if any non-ignored error occurred.
func runHostTasks(host inventory.Host, serviceTasks []Task, handlers []Handler, opts RunOptions, vars map[string]interface{}) bool {
	notified := make(map[string]bool)
	failed := false

	for _, task := range serviceTasks {
		if !matchesTags(task.Tags, opts.Tags, opts.SkipTags) {
			continue
		}

		fmt.Printf("  TASK [%s]\n", task.Name)
		if err := executeTask(task, host, opts, vars); err != nil {
			fmt.Printf("  failed: %v\n", err)
			if !task.IgnoreErrors {
				failed = true
				if opts.FailFast {
					return failed
				}
			}
		} else {
			fmt.Println("  ok")
			if task.Notify != "" {
				notified[task.Notify] = true
			}
		}
	}

	// Fire notified handlers (each at most once per host)
	for _, h := range handlers {
		if !notified[h.Name] {
			continue
		}
		fmt.Printf("  HANDLER [%s]\n", h.Name)
		hTask := Task{Name: h.Name, Command: h.Command}
		if err := executeTask(hTask, host, opts, vars); err != nil {
			fmt.Printf("  handler failed: %v\n", err)
		} else {
			fmt.Println("  ok")
		}
	}

	return failed
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// RunPlaybook executes a full playbook against an inventory via SSH or locally.
// Pass inv=nil only when opts.RunLocally=true.
func RunPlaybook(playbook Playbook, inv *inventory.Inventory, opts RunOptions) error {
	if opts.ServicesPath == "" {
		opts.ServicesPath = DefaultServicesPath
	}
	if opts.Forks <= 0 {
		opts.Forks = 5
	}

	overallFailed := false

	for _, play := range playbook {
		if !matchesTags(play.Tags, opts.Tags, opts.SkipTags) {
			continue
		}

		fmt.Printf("PLAY [%s] **************************************************\n", play.Name)

		var hosts []inventory.Host
		var groupVars map[string]interface{}

		if opts.RunLocally {
			hosts = []inventory.Host{{Address: "localhost"}}
		} else {
			var ok bool
			hosts, ok = inv.Hosts[play.Hosts]
			if !ok {
				fmt.Printf("No hosts found for group: %s\n", play.Hosts)
				continue
			}
			groupVars = hostVarsToInterface(inv.GroupVars[play.Hosts])
		}

		for _, service := range play.Services {
			serviceTasks, err := LoadServiceTasks(opts.ServicesPath, service.ServiceName)
			if err != nil {
				fmt.Printf("Error loading service [%s]: %v\n", service.ServiceName, err)
				continue
			}

			sem := make(chan struct{}, opts.Forks)
			var wg sync.WaitGroup
			var mu sync.Mutex

			for _, host := range hosts {
				host := host // capture loop variable
				wg.Add(1)
				sem <- struct{}{}
				go func(h inventory.Host) {
					defer wg.Done()
					defer func() { <-sem }()

					fmt.Printf("HOST [%s] ---\n", h.Address)

					vars := mergeVars(play.Vars, groupVars, hostVarsToInterface(h.Vars))
					if runHostTasks(h, serviceTasks, play.Handlers, opts, vars) {
						mu.Lock()
						overallFailed = true
						mu.Unlock()
					}
				}(host)
			}
			wg.Wait()

			if overallFailed && opts.FailFast {
				return fmt.Errorf("playbook aborted: fail_fast triggered")
			}
		}
	}

	if overallFailed {
		return fmt.Errorf("playbook completed with errors")
	}
	return nil
}

// RunAdHocCommand runs a single command or script against all hosts in a group.
func RunAdHocCommand(inv *inventory.Inventory, group, command string, opts RunOptions) error {
	hosts, ok := inv.Hosts[group]
	if !ok {
		return fmt.Errorf("no hosts found for group: %s", group)
	}
	if opts.Forks <= 0 {
		opts.Forks = 5
	}

	task := Task{Name: "ad hoc", Command: command}
	sem := make(chan struct{}, opts.Forks)
	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := false

	for _, host := range hosts {
		host := host
		wg.Add(1)
		sem <- struct{}{}
		go func(h inventory.Host) {
			defer wg.Done()
			defer func() { <-sem }()

			fmt.Printf("TASK [ad hoc] on host [%s] ***********************************\n", h.Address)
			if err := executeTask(task, h, opts, nil); err != nil {
				fmt.Printf("failed: %v\n", err)
				mu.Lock()
				failed = true
				mu.Unlock()
			} else {
				fmt.Println("ok")
			}
		}(host)
	}
	wg.Wait()

	if failed {
		return fmt.Errorf("ad hoc command failed on one or more hosts")
	}
	return nil
}

// RunLocalAdHocCommand runs a single command or script locally without SSH or inventory.
func RunLocalAdHocCommand(command string) error {
	fmt.Printf("TASK [local ad hoc] ***********************************\n")
	task := Task{Name: "local ad hoc", Command: command}
	h := inventory.Host{Address: "localhost"}
	opts := RunOptions{RunLocally: true}
	if err := executeTask(task, h, opts, nil); err != nil {
		fmt.Printf("failed: %v\n", err)
		return err
	}
	fmt.Println("ok")
	return nil
}

// ---------------------------------------------------------------------------
// Local execution helpers
// ---------------------------------------------------------------------------

func runLocalCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command execution failed: %v, output: %s", err, output)
	}
	fmt.Printf("Output:\n%s\n", output)
	return nil
}

func runLocalScript(scriptPath string) error {
	cmd := exec.Command("sh", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %v, output: %s", err, output)
	}
	fmt.Printf("Output:\n%s\n", output)
	return nil
}

func copyLocal(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	fmt.Printf("Copied %s -> %s\n", src, dest)
	return nil
}
