package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"for/pkg/inventory"
	"for/pkg/ssh"
	"for/pkg/utils"
	"gopkg.in/yaml.v2"
)

// DefaultServicesPath is the default base directory for service task files.
const DefaultServicesPath = "services"

type Playbook []Play

type Play struct {
	Name     string    `yaml:"name"`
	Hosts    string    `yaml:"hosts"`
	Services []Service `yaml:"services"`
}

type Service struct {
	ServiceName string `yaml:"service"`
}

type Task struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
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

// executeTask runs a single task either locally or on a remote host via SSH.
func executeTask(task Task, host, sshUser, sshKeyPath string, sshPort int, runLocally bool) error {
	if runLocally {
		if utils.IsScript(task.Command) {
			return runLocalScript(task.Command)
		}
		return runLocalCommand(task.Command)
	}
	if utils.IsScript(task.Command) {
		return ssh.RunScript(host, task.Command, sshUser, sshKeyPath, sshPort)
	}
	return ssh.RunCommand(host, task.Command, sshUser, sshKeyPath, sshPort)
}

// RunPlaybook executes a full playbook against an inventory via SSH or locally.
// Pass inv=nil only when runLocally=true.
func RunPlaybook(playbook Playbook, inv *inventory.Inventory, sshUser, sshKeyPath string, sshPort int, servicesPath string, runLocally bool) error {
	if servicesPath == "" {
		servicesPath = DefaultServicesPath
	}
	for _, play := range playbook {
		fmt.Printf("PLAY [%s] **************************************************\n", play.Name)

		var hosts []string
		if !runLocally {
			var ok bool
			hosts, ok = inv.Hosts[play.Hosts]
			if !ok {
				fmt.Printf("No hosts found for group: %s\n", play.Hosts)
				continue
			}
		}

		for _, service := range play.Services {
			serviceTasks, err := LoadServiceTasks(servicesPath, service.ServiceName)
			if err != nil {
				fmt.Printf("Error loading service [%s]: %v\n", service.ServiceName, err)
				continue
			}

			if runLocally {
				for _, task := range serviceTasks {
					fmt.Printf("TASK [%s]\n", task.Name)
					if err := executeTask(task, "", "", "", 0, true); err != nil {
						fmt.Printf("failed: %v\n", err)
					} else {
						fmt.Println("ok")
					}
				}
			} else {
				for _, host := range hosts {
					fmt.Printf("TASK [%s] on host [%s] ***********************************\n", play.Name, host)
					for _, task := range serviceTasks {
						fmt.Printf("TASK [%s]\n", task.Name)
						if err := executeTask(task, host, sshUser, sshKeyPath, sshPort, false); err != nil {
							fmt.Printf("failed: %v\n", err)
						} else {
							fmt.Println("ok")
						}
					}
				}
			}
		}
	}
	return nil
}

// RunAdHocCommand runs a single command or script against all hosts in a group.
func RunAdHocCommand(inv *inventory.Inventory, group, command, sshUser, sshKeyPath string, sshPort int, runLocally bool) error {
	hosts, ok := inv.Hosts[group]
	if !ok {
		return fmt.Errorf("no hosts found for group: %s", group)
	}

	task := Task{Name: "Ad hoc command", Command: command}
	for _, host := range hosts {
		fmt.Printf("TASK [Ad hoc command] on host [%s] ***********************************\n", host)
		if err := executeTask(task, host, sshUser, sshKeyPath, sshPort, runLocally); err != nil {
			fmt.Printf("failed: %v\n", err)
		} else {
			fmt.Println("ok")
		}
	}
	return nil
}

// RunLocalAdHocCommand runs a single command or script locally without SSH or inventory.
func RunLocalAdHocCommand(command string) error {
	fmt.Printf("TASK [Local ad hoc command] ***********************************\n")
	task := Task{Name: "Local ad hoc command", Command: command}
	if err := executeTask(task, "", "", "", 0, true); err != nil {
		fmt.Printf("failed: %v\n", err)
		return err
	}
	fmt.Println("ok")
	return nil
}

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
