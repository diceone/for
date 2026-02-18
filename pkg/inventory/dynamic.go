package inventory

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// DynamicGroup is one entry in the JSON produced by a dynamic inventory script.
type DynamicGroup struct {
	Hosts []string          `json:"hosts"`
	Vars  map[string]string `json:"vars"`
}

// LoadDynamic executes a script and parses its stdout as a JSON inventory.
//
// Expected JSON format:
//
//	{
//	  "webservers": {
//	    "hosts": ["192.168.1.10", "192.168.1.11"],
//	    "vars":  {"env": "production"}
//	  },
//	  "dbservers": {
//	    "hosts": ["192.168.1.20"]
//	  }
//	}
func LoadDynamic(script string) (*Inventory, error) {
	out, err := exec.Command(script).Output()
	if err != nil {
		return nil, fmt.Errorf("dynamic inventory script %q: %w", script, err)
	}

	var raw map[string]DynamicGroup
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing dynamic inventory JSON: %w", err)
	}

	inv := &Inventory{
		Hosts:     make(map[string][]Host),
		GroupVars: make(map[string]map[string]string),
	}

	for group, data := range raw {
		for _, addr := range data.Hosts {
			inv.Hosts[group] = append(inv.Hosts[group], Host{
				Address: addr,
				Vars:    make(map[string]string),
			})
		}
		if len(data.Vars) > 0 {
			inv.GroupVars[group] = data.Vars
		}
	}
	return inv, nil
}
