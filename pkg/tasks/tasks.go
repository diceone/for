package tasks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"for/pkg/facts"
	"for/pkg/inventory"
	"for/pkg/printer"
	"for/pkg/ssh"
	"for/pkg/utils"
	"gopkg.in/yaml.v3"
)

// DefaultServicesPath is the default base directory for service task files.
const DefaultServicesPath = "services"

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

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

// CopyTask describes a local to remote file copy.
type CopyTask struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

type Task struct {
	Name         string        `yaml:"name"`
	Command      string        `yaml:"command"`
	Copy         *CopyTask     `yaml:"copy"`
	IgnoreErrors bool          `yaml:"ignore_errors"`
	Tags         []string      `yaml:"tags"`
	Notify       string        `yaml:"notify"`
	When         string        `yaml:"when"`
	WithItems    []interface{} `yaml:"with_items"`
	Timeout      string        `yaml:"timeout"`
	Retries      int           `yaml:"retries"`
	Delay        string        `yaml:"delay"`
	Register     string        `yaml:"register"`
	ChangedWhen  string        `yaml:"changed_when"`
}

// TaskResult captures the outcome of a single task execution.
type TaskResult struct {
	Output  string
	Changed bool
	Failed  bool
	RC      int
}

// ServiceMeta declares role/service dependencies.
type ServiceMeta struct {
	Dependencies []string `yaml:"dependencies"`
}

// RunOptions consolidates all execution parameters.
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
	SSHPool        *ssh.Pool
	GatherFacts    bool
}

// ---------------------------------------------------------------------------
// Loaders
// ---------------------------------------------------------------------------

func LoadTasks(file string) (Playbook, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var playbook Playbook
	return playbook, yaml.Unmarshal(data, &playbook)
}

// LoadServiceMeta loads meta/main.yaml for a service (role dependencies).
func LoadServiceMeta(servicesPath, serviceName string) (*ServiceMeta, error) {
	if servicesPath == "" {
		servicesPath = DefaultServicesPath
	}
	metaPath := filepath.Join(servicesPath, serviceName, "meta", "main.yaml")
	data, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return &ServiceMeta{}, nil
	}
	if err != nil {
		return nil, err
	}
	var meta ServiceMeta
	return &meta, yaml.Unmarshal(data, &meta)
}

// LoadServiceTasks loads the task list for a named service.
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
	return serviceTasks, yaml.Unmarshal(data, &serviceTasks)
}

// LoadServiceTasksWithDeps loads tasks for a service and all its dependencies.
func LoadServiceTasksWithDeps(servicesPath, serviceName string) ([]Task, error) {
	return loadWithDeps(servicesPath, serviceName, map[string]bool{})
}

func loadWithDeps(servicesPath, name string, visited map[string]bool) ([]Task, error) {
	if visited[name] {
		return nil, nil
	}
	visited[name] = true

	meta, err := LoadServiceMeta(servicesPath, name)
	if err != nil {
		return nil, err
	}

	var all []Task
	for _, dep := range meta.Dependencies {
		depTasks, err := loadWithDeps(servicesPath, dep, visited)
		if err != nil {
			return nil, fmt.Errorf("dependency %q: %w", dep, err)
		}
		all = append(all, depTasks...)
	}

	own, err := LoadServiceTasks(servicesPath, name)
	if err != nil {
		return nil, err
	}
	return append(all, own...), nil
}

// ---------------------------------------------------------------------------
// Tag helpers
// ---------------------------------------------------------------------------

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
	return false
}

// ---------------------------------------------------------------------------
// Template helpers
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

// evaluateCondition renders the when expression and returns true unless result is falsy.
func evaluateCondition(when string, vars map[string]interface{}) (bool, error) {
	if when == "" {
		return true, nil
	}
	result, err := expandVars(when, vars)
	if err != nil {
		return false, err
	}
	r := strings.TrimSpace(strings.ToLower(result))
	return r != "" && r != "false" && r != "0" && r != "no", nil
}

func isTruthy(expr string, vars map[string]interface{}) bool {
	result, err := expandVars(expr, vars)
	if err != nil {
		return false
	}
	r := strings.TrimSpace(strings.ToLower(result))
	return r != "" && r != "false" && r != "0" && r != "no"
}

// ---------------------------------------------------------------------------
// SSH config builder
// ---------------------------------------------------------------------------

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
// Low-level execution with timeout and retry
// ---------------------------------------------------------------------------

func runOnce(host inventory.Host, task Task, opts RunOptions, vars map[string]interface{}) (TaskResult, error) {
	cmd, err := expandVars(task.Command, vars)
	if err != nil {
		return TaskResult{Failed: true}, fmt.Errorf("template: %w", err)
	}

	if opts.DryRun {
		if task.Copy != nil {
			printer.DryRun(fmt.Sprintf("COPY %s -> %s:%s", task.Copy.Src, host.Address, task.Copy.Dest))
		} else {
			printer.DryRun(fmt.Sprintf("CMD %s", cmd))
		}
		return TaskResult{}, nil
	}

	if task.Copy != nil {
		if opts.RunLocally {
			err = copyLocal(task.Copy.Src, task.Copy.Dest)
		} else {
			err = ssh.CopyFile(host.Address, task.Copy.Src, task.Copy.Dest, sshConfigFor(host, opts))
		}
		if err != nil {
			return TaskResult{Failed: true, RC: 1}, err
		}
		return TaskResult{Changed: true}, nil
	}

	var output string
	if opts.RunLocally {
		if utils.IsScript(cmd) {
			output, err = runLocalScriptOutput(cmd)
		} else {
			output, err = runLocalCommandOutput(cmd)
		}
	} else {
		sshCfg := sshConfigFor(host, opts)
		if opts.SSHPool != nil {
			output, err = opts.SSHPool.RunCommandOutput(host.Address, cmd, sshCfg)
		} else if utils.IsScript(cmd) {
			output, err = runRemoteScript(host.Address, cmd, sshCfg)
		} else {
			output, err = ssh.RunCommandOutput(host.Address, cmd, sshCfg)
		}
	}

	res := TaskResult{Output: output}
	if err != nil {
		res.Failed = true
		res.RC = 1
	}
	if task.ChangedWhen != "" {
		localVars := mergeVars(vars, map[string]interface{}{"output": output})
		res.Changed = isTruthy(task.ChangedWhen, localVars)
	} else {
		res.Changed = !res.Failed
	}
	return res, err
}

func runWithTimeout(timeout string, fn func() (TaskResult, error)) (TaskResult, error) {
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return TaskResult{Failed: true}, fmt.Errorf("invalid timeout %q: %w", timeout, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	type pair struct {
		r TaskResult
		e error
	}
	ch := make(chan pair, 1)
	go func() {
		r, e := fn()
		ch <- pair{r, e}
	}()
	select {
	case <-ctx.Done():
		return TaskResult{Failed: true}, fmt.Errorf("timed out after %s", timeout)
	case p := <-ch:
		return p.r, p.e
	}
}

func runWithRetry(retries int, delay string, fn func() (TaskResult, error)) (TaskResult, error) {
	var d time.Duration
	if delay != "" {
		var err error
		d, err = time.ParseDuration(delay)
		if err != nil {
			return TaskResult{Failed: true}, fmt.Errorf("invalid delay %q: %w", delay, err)
		}
	}
	var (
		res TaskResult
		err error
	)
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			fmt.Printf("    retry %d/%d\n", attempt, retries)
			if d > 0 {
				time.Sleep(d)
			}
		}
		res, err = fn()
		if err == nil {
			return res, nil
		}
	}
	return res, err
}

// executeTask applies when/with_items/timeout/retry logic and delegates to runOnce.
func executeTask(task Task, host inventory.Host, opts RunOptions, vars map[string]interface{}) (TaskResult, error) {
	ok, err := evaluateCondition(task.When, vars)
	if err != nil {
		return TaskResult{Failed: true}, fmt.Errorf("when eval: %w", err)
	}
	if !ok {
		return TaskResult{}, nil
	}

	run := func(loopVars map[string]interface{}) (TaskResult, error) {
		merged := mergeVars(vars, loopVars)
		fn := func() (TaskResult, error) {
			return runOnce(host, task, opts, merged)
		}
		if task.Timeout != "" {
			fn2 := fn
			fn = func() (TaskResult, error) {
				return runWithTimeout(task.Timeout, fn2)
			}
		}
		if task.Retries > 0 {
			return runWithRetry(task.Retries, task.Delay, fn)
		}
		return fn()
	}

	if len(task.WithItems) > 0 {
		combined := TaskResult{}
		for _, item := range task.WithItems {
			res, err := run(map[string]interface{}{"item": item})
			combined.Output += res.Output
			if res.Changed {
				combined.Changed = true
			}
			if err != nil {
				combined.Failed = true
				if !task.IgnoreErrors {
					return combined, err
				}
			}
		}
		return combined, nil
	}
	return run(nil)
}

// ---------------------------------------------------------------------------
// Per-host runner
// ---------------------------------------------------------------------------

func runHostTasks(host inventory.Host, serviceTasks []Task, handlers []Handler, opts RunOptions, vars map[string]interface{}) printer.HostSummary {
	notified := make(map[string]bool)
	summary := printer.HostSummary{Host: host.Address}

	for _, task := range serviceTasks {
		if !matchesTags(task.Tags, opts.Tags, opts.SkipTags) {
			summary.Skipped++
			continue
		}

		printer.TaskHeader(task.Name)

		res, err := executeTask(task, host, opts, vars)

		if task.Register != "" && vars != nil {
			vars[task.Register] = res.Output
			printer.RegisterNote(task.Register, res.Output)
		}

		switch {
		case err != nil:
			if task.IgnoreErrors {
				printer.Ignored(host.Address, err)
				summary.Ignored++
			} else {
				printer.Failed(host.Address, err)
				summary.Failed++
				if opts.FailFast {
					return summary
				}
			}
		case !res.Changed && !res.Failed && task.When != "" && res.Output == "":
			printer.Skipped(host.Address)
			summary.Skipped++
		case res.Changed:
			printer.Changed(host.Address, res.Output)
			summary.Changed++
			if task.Notify != "" {
				notified[task.Notify] = true
			}
		default:
			printer.OK(host.Address, res.Output)
			summary.OK++
			if task.Notify != "" {
				notified[task.Notify] = true
			}
		}
	}

	for _, h := range handlers {
		if !notified[h.Name] {
			continue
		}
		printer.HandlerHeader(h.Name)
		hTask := Task{Name: h.Name, Command: h.Command}
		res, err := executeTask(hTask, host, opts, vars)
		if err != nil {
			printer.Failed(host.Address, err)
			summary.Failed++
		} else if res.Changed {
			printer.Changed(host.Address, res.Output)
			summary.Changed++
		} else {
			printer.OK(host.Address, res.Output)
			summary.OK++
		}
	}

	return summary
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// RunPlaybook executes a full playbook and prints a PLAY RECAP.
func RunPlaybook(playbook Playbook, inv *inventory.Inventory, opts RunOptions) error {
	if opts.ServicesPath == "" {
		opts.ServicesPath = DefaultServicesPath
	}
	if opts.Forks <= 0 {
		opts.Forks = 5
	}

	overallFailed := false
	var recapMu sync.Mutex
	allSummaries := make(map[string]printer.HostSummary)

	ownPool := false
	if opts.SSHPool == nil && !opts.RunLocally {
		opts.SSHPool = ssh.NewPool()
		ownPool = true
	}
	if ownPool {
		defer opts.SSHPool.Close()
	}

	for _, play := range playbook {
		if !matchesTags(play.Tags, opts.Tags, opts.SkipTags) {
			continue
		}

		printer.PlayHeader(play.Name)

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

		var localFacts map[string]interface{}
		if opts.GatherFacts && opts.RunLocally {
			localFacts = map[string]interface{}(facts.GatherLocal())
		}

		for _, service := range play.Services {
			serviceTasks, err := LoadServiceTasksWithDeps(opts.ServicesPath, service.ServiceName)
			if err != nil {
				fmt.Printf("Error loading service [%s]: %v\n", service.ServiceName, err)
				continue
			}

			sem := make(chan struct{}, opts.Forks)
			var wg sync.WaitGroup

			for _, host := range hosts {
				host := host
				wg.Add(1)
				sem <- struct{}{}
				go func(h inventory.Host) {
					defer wg.Done()
					defer func() { <-sem }()

					printer.HostHeader(h.Address)

					hostFacts := localFacts
					if opts.GatherFacts && !opts.RunLocally {
						sshCfg := sshConfigFor(h, opts)
						hostFacts = map[string]interface{}(facts.GatherRemote(h, sshCfg))
					}

					vars := mergeVars(play.Vars, groupVars, hostVarsToInterface(h.Vars), hostFacts)
					sum := runHostTasks(h, serviceTasks, play.Handlers, opts, vars)

					recapMu.Lock()
					prev := allSummaries[h.Address]
					prev.Host = h.Address
					prev.OK += sum.OK
					prev.Changed += sum.Changed
					prev.Failed += sum.Failed
					prev.Skipped += sum.Skipped
					prev.Ignored += sum.Ignored
					allSummaries[h.Address] = prev
					if sum.Failed > 0 {
						overallFailed = true
					}
					recapMu.Unlock()
				}(host)
			}
			wg.Wait()

			if overallFailed && opts.FailFast {
				break
			}
		}

		if overallFailed && opts.FailFast {
			break
		}
	}

	summaries := make([]printer.HostSummary, 0, len(allSummaries))
	for _, s := range allSummaries {
		summaries = append(summaries, s)
	}
	printer.Recap(summaries)

	if overallFailed {
		return fmt.Errorf("playbook completed with errors")
	}
	return nil
}

// RunAdHocCommand runs a single command against all hosts in a group.
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
			printer.TaskHeader("ad hoc: "+command)
			printer.HostHeader(h.Address)
			res, err := executeTask(task, h, opts, nil)
			if err != nil {
				printer.Failed(h.Address, err)
				mu.Lock()
				failed = true
				mu.Unlock()
			} else {
				printer.OK(h.Address, res.Output)
			}
		}(host)
	}
	wg.Wait()

	if failed {
		return fmt.Errorf("ad hoc command failed on one or more hosts")
	}
	return nil
}

// RunLocalAdHocCommand runs a single command locally.
func RunLocalAdHocCommand(command string) error {
	printer.TaskHeader("local ad hoc: "+command)
	task := Task{Name: "local ad hoc", Command: command}
	h := inventory.Host{Address: "localhost"}
	opts := RunOptions{RunLocally: true}
	res, err := executeTask(task, h, opts, nil)
	if err != nil {
		printer.Failed("localhost", err)
		return err
	}
	printer.OK("localhost", res.Output)
	return nil
}

// ---------------------------------------------------------------------------
// Local execution helpers
// ---------------------------------------------------------------------------

func runLocalCommandOutput(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runLocalScriptOutput(scriptPath string) (string, error) {
	cmd := exec.Command("sh", scriptPath)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runRemoteScript(host, scriptPath string, cfg ssh.Config) (string, error) {
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return ssh.RunCommandOutput(host, string(script), cfg)
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
