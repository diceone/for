package utils

import (
	"os"
	"testing"
)

func TestIsScript_NonExistentShFile(t *testing.T) {
	// Non-existent path with .sh extension → false (file must exist)
	if IsScript("/nonexistent/path/script.sh") {
		t.Error("expected false for non-existent script")
	}
}

func TestIsScript_ExistingShFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test_*.sh")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	f.Close()
	if !IsScript(f.Name()) {
		t.Errorf("expected true for existing .sh file: %s", f.Name())
	}
}

func TestIsScript_PlainCommand(t *testing.T) {
	if IsScript("apt-get install nginx") {
		t.Error("expected false for a plain shell command")
	}
}

func TestIsScript_NoExtension(t *testing.T) {
	if IsScript("myscript") {
		t.Error("expected false for file without script extension")
	}
}
