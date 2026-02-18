// Package facts gathers system information from local or remote hosts and
// exposes them as template variables (e.g. {{ .os }}, {{ .arch }}).
package facts

import (
	"os/exec"
	"runtime"
	"strings"

	"for/pkg/inventory"
	"for/pkg/ssh"
)

// Facts is a map from fact name to value, directly usable as template data.
type Facts map[string]interface{}

// GatherLocal collects facts from the local machine.
func GatherLocal() Facts {
	f := Facts{
		"os":                 runtime.GOOS,
		"arch":               runtime.GOARCH,
		"inventory_hostname": "localhost",
	}
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		f["kernel"] = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("hostname").Output(); err == nil {
		f["hostname"] = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("hostname", "-f").Output(); err == nil {
		f["fqdn"] = strings.TrimSpace(string(out))
	}
	return f
}

// GatherRemote collects facts from a remote host via SSH.
// Facts that cannot be collected are silently omitted.
func GatherRemote(host inventory.Host, cfg ssh.Config) Facts {
	f := Facts{
		"inventory_hostname": host.Address,
	}

	cmds := map[string]string{
		"os":             "uname -s | tr '[:upper:]' '[:lower:]'",
		"arch":           "uname -m",
		"kernel":         "uname -r",
		"hostname":       "hostname 2>/dev/null || echo " + host.Address,
		"fqdn":           "hostname -f 2>/dev/null || hostname 2>/dev/null || echo " + host.Address,
		"distro":         "grep ^ID= /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '\"' || echo unknown",
		"distro_version": "grep ^VERSION_ID= /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '\"' || echo unknown",
		"cpu_count":      "nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 1",
		"total_memory":   "free -m 2>/dev/null | awk '/^Mem:/{print $2}' || echo unknown",
	}

	for key, cmd := range cmds {
		if out, err := ssh.RunCommandOutput(host.Address, cmd, cfg); err == nil {
			f[key] = strings.TrimSpace(out)
		}
	}
	return f
}
