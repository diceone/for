package inventory

import (
	"os"
	"testing"
)

func TestLoadInventory_SkipsCommentsAndBlanks(t *testing.T) {
	f := writeTempFile(t, `# this is a comment

[webservers]
192.168.1.10
# another comment
192.168.1.11
`)
	inv, err := LoadInventory(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hosts := inv.Hosts["webservers"]
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Address != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10, got %s", hosts[0].Address)
	}
}

func TestLoadInventory_HostVars(t *testing.T) {
	f := writeTempFile(t, `
[webservers]
192.168.1.10 ssh_port=2222 ansible_user=admin
`)
	inv, err := LoadInventory(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h := inv.Hosts["webservers"][0]
	if h.Address != "192.168.1.10" {
		t.Errorf("expected address 192.168.1.10, got %s", h.Address)
	}
	if h.Vars["ssh_port"] != "2222" {
		t.Errorf("expected ssh_port=2222, got %s", h.Vars["ssh_port"])
	}
	if h.Vars["ansible_user"] != "admin" {
		t.Errorf("expected ansible_user=admin, got %s", h.Vars["ansible_user"])
	}
}

func TestLoadInventory_GroupVars(t *testing.T) {
	f := writeTempFile(t, `
[webservers]
192.168.1.10

[webservers:vars]
env=production
version=1.2.3
`)
	inv, err := LoadInventory(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.GroupVars["webservers"]["env"] != "production" {
		t.Errorf("expected env=production, got %s", inv.GroupVars["webservers"]["env"])
	}
	if inv.GroupVars["webservers"]["version"] != "1.2.3" {
		t.Errorf("expected version=1.2.3, got %s", inv.GroupVars["webservers"]["version"])
	}
}

func TestLoadInventory_MultipleGroups(t *testing.T) {
	f := writeTempFile(t, `
[webservers]
192.168.1.10
192.168.1.11

[dbservers]
192.168.1.20
`)
	inv, err := LoadInventory(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.Hosts["webservers"]) != 2 {
		t.Errorf("expected 2 webservers, got %d", len(inv.Hosts["webservers"]))
	}
	if len(inv.Hosts["dbservers"]) != 1 {
		t.Errorf("expected 1 dbserver, got %d", len(inv.Hosts["dbservers"]))
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "inventory_*.ini")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
