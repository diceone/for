package inventory

import (
	"bufio"
	"os"
	"strings"
)

// Host represents a single target host with optional per-host variables.
type Host struct {
	Address string
	Vars    map[string]string
}

// Inventory holds parsed host groups and group-level variables.
type Inventory struct {
	Hosts     map[string][]Host
	GroupVars map[string]map[string]string
}

func LoadInventory(file string) (*Inventory, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	inv := &Inventory{
		Hosts:     make(map[string][]Host),
		GroupVars: make(map[string]map[string]string),
	}

	scanner := bufio.NewScanner(f)
	var group string
	var isVarsSection bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inner := line[1 : len(line)-1]
			if strings.HasSuffix(inner, ":vars") {
				group = strings.TrimSuffix(inner, ":vars")
				isVarsSection = true
			} else {
				group = inner
				isVarsSection = false
			}
		} else if group != "" {
			if isVarsSection {
				if inv.GroupVars[group] == nil {
					inv.GroupVars[group] = make(map[string]string)
				}
				key, val, _ := strings.Cut(line, "=")
				inv.GroupVars[group][strings.TrimSpace(key)] = strings.TrimSpace(val)
			} else {
				inv.Hosts[group] = append(inv.Hosts[group], parseHostLine(line))
			}
		}
	}

	return inv, scanner.Err()
}

// parseHostLine parses a host entry such as:
//
//	192.168.1.10 ssh_port=2222 ansible_user=admin
func parseHostLine(line string) Host {
	parts := strings.Fields(line)
	host := Host{
		Address: parts[0],
		Vars:    make(map[string]string),
	}
	for _, part := range parts[1:] {
		key, val, _ := strings.Cut(part, "=")
		host.Vars[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return host
}
