package tasks

import (
    "fmt"
    "for/pkg/ssh"
    "for/pkg/inventory"
    "for/pkg/utils"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "os/exec"
    "path/filepath"
)

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
    data, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, err
    }

    var playbook Playbook
    err = yaml.Unmarshal(data, &playbook)
    if err != nil {
        return nil, err
    }

    return playbook, nil
}

func LoadServiceTasks(serviceName string) ([]Task, error) {
    serviceFilePath := filepath.Join("services", serviceName, "tasks", "main.yaml")
    data, err := ioutil.ReadFile(serviceFilePath)
    if err != nil {
        return nil, err
    }

    var tasks []Task
    err = yaml.Unmarshal(data, &tasks)
    if err != nil {
        return nil, err
    }

    return tasks, nil
}

func RunPlaybook(playbook Playbook, inv *inventory.Inventory, sshUser, sshKeyPath string, runLocally bool) {
    for _, play := range playbook {
        fmt.Printf("PLAY [%s] **************************************************\n", play.Name)
        hosts, ok := inv.Hosts[play.Hosts]
        if !ok {
            fmt.Printf("No hosts found for group: %s\n", play.Hosts)
            continue
        }

        for _, service := range play.Services {
            tasks, err := LoadServiceTasks(service.ServiceName)
            if err != nil {
                fmt.Printf("Error loading service [%s]: %v\n", service.ServiceName, err)
                continue
            }

            for _, host := range hosts {
                fmt.Printf("TASK [%s] on host [%s] ***********************************\n", play.Name, host)
                for _, task := range tasks {
                    fmt.Printf("TASK [%s]\n", task.Name)
                    if runLocally {
                        if utils.IsScript(task.Command) {
                            err := runLocalScript(task.Command)
                            if err != nil {
                                fmt.Printf("failed: %v\n", err)
                            } else {
                                fmt.Println("ok")
                            }
                        } else {
                            err := runLocalCommand(task.Command)
                            if err != nil {
                                fmt.Printf("failed: %v\n", err)
                            } else {
                                fmt.Println("ok")
                            }
                        }
                    } else {
                        if utils.IsScript(task.Command) {
                            err := ssh.RunScript(host, task.Command, sshUser, sshKeyPath)
                            if err != nil {
                                fmt.Printf("failed: %v\n", err)
                            } else {
                                fmt.Println("ok")
                            }
                        } else {
                            err := ssh.RunCommand(host, task.Command, sshUser, sshKeyPath)
                            if err != nil {
                                fmt.Printf("failed: %v\n", err)
                            } else {
                                fmt.Println("ok")
                            }
                        }
                    }
                }
            }
        }
    }
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

func RunAdHocCommand(inv *inventory.Inventory, group, command, sshUser, sshKeyPath string, runLocally bool) {
    hosts, ok := inv.Hosts[group]
    if !ok {
        fmt.Printf("No hosts found for group: %s\n", group)
        return
    }

    for _, host := range hosts {
        fmt.Printf("TASK [Ad hoc command] on host [%s] ***********************************\n", host)
        if runLocally {
            if utils.IsScript(command) {
                err := runLocalScript(command)
                if err != nil {
                    fmt.Printf("failed: %v\n", err)
                } else {
                    fmt.Println("ok")
                }
            } else {
                err := runLocalCommand(command)
                if err != nil {
                    fmt.Printf("failed: %v\n", err)
                } else {
                    fmt.Println("ok")
                }
            }
        } else {
            if utils.IsScript(command) {
                err := ssh.RunScript(host, command, sshUser, sshKeyPath)
                if err != nil {
                    fmt.Printf("failed: %v\n", err)
                } else {
                    fmt.Println("ok")
                }
            } else {
                err := ssh.RunCommand(host, command, sshUser, sshKeyPath)
                if err != nil {
                    fmt.Printf("failed: %v\n", err)
                } else {
                    fmt.Println("ok")
                }
            }
        }
    }
}
