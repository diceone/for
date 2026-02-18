package inventory

import (
	"bufio"
	"os"
	"strings"
)

type Inventory struct {
	Hosts map[string][]string
}

func LoadInventory(file string) (*Inventory, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	inv := &Inventory{Hosts: make(map[string][]string)}
	scanner := bufio.NewScanner(f)
	var group string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			group = line[1 : len(line)-1]
		} else if group != "" {
			inv.Hosts[group] = append(inv.Hosts[group], line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return inv, nil
}
