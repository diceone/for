package utils

import (
    "os"
    "path/filepath"
)

// IsScript checks if the given command is a script file based on its extension and existence.
func IsScript(command string) bool {
    ext := filepath.Ext(command)
    if ext == ".sh" || ext == ".bash" || ext == ".zsh" {
        if _, err := os.Stat(command); err == nil {
            return true
        }
    }
    return false
}
