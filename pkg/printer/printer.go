// Package printer provides coloured, structured console output and PLAY RECAP.
package printer

import (
	"fmt"
	"os"
	"strings"
)

// ANSI colour codes.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
)

// ColorsEnabled controls ANSI output. Auto-detected from stdout; can be overridden.
var ColorsEnabled = isTerminal()

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func c(color, s string) string {
	if !ColorsEnabled {
		return s
	}
	return color + s + ansiReset
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// HostSummary tracks task execution counts for one host across a full playbook run.
type HostSummary struct {
	Host    string
	OK      int
	Changed int
	Failed  int
	Skipped int
	Ignored int
}

// PlayHeader prints the PLAY banner.
func PlayHeader(name string) {
	sep := strings.Repeat("*", max(0, 72-len(name)-8))
	fmt.Printf("\n%s [%s] %s\n", c(ansiBold+ansiBlue, "PLAY"), c(ansiBold, name), sep)
}

// TaskHeader prints the TASK banner.
func TaskHeader(name string) {
	sep := strings.Repeat("-", max(0, 72-len(name)-8))
	fmt.Printf("\n%s [%s] %s\n", c(ansiBold, "TASK"), name, sep)
}

// HandlerHeader prints the HANDLER banner.
func HandlerHeader(name string) {
	sep := strings.Repeat("-", max(0, 72-len(name)-11))
	fmt.Printf("\n%s [%s] %s\n", c(ansiBold, "HANDLER"), name, sep)
}

// HostHeader prints a host separator line.
func HostHeader(host string) {
	fmt.Printf("\n%s\n", c(ansiCyan, "  HOST ["+host+"]"))
}

// OK prints an ok result line and optional output.
func OK(host, output string) {
	fmt.Printf("  %s: [%s]\n", c(ansiGreen, "ok"), host)
	if strings.TrimSpace(output) != "" {
		Output("stdout", output)
	}
}

// Changed prints a changed result line and optional output.
func Changed(host, output string) {
	fmt.Printf("  %s: [%s]\n", c(ansiYellow, "changed"), host)
	if strings.TrimSpace(output) != "" {
		Output("stdout", output)
	}
}

// Failed prints a failed result line.
func Failed(host string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	fmt.Printf("  %s: [%s]\n", c(ansiRed, "FAILED"), host)
	if msg != "" {
		fmt.Printf("  %s\n", strings.TrimSpace(msg))
	}
}

// Ignored prints an ignored-error result line.
func Ignored(host string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	fmt.Printf("  %s: [%s] (ignored)\n", c(ansiYellow, "failed"), host)
	if msg != "" {
		fmt.Printf("  %s\n", strings.TrimSpace(msg))
	}
}

// Skipped prints a skipped result line.
func Skipped(host string) {
	fmt.Printf("  %s: [%s]\n", c(ansiCyan, "skipping"), host)
}

// DryRun prints a dry-run line for a command or copy.
func DryRun(msg string) {
	fmt.Printf("  %s %s\n", c(ansiCyan, "[dry-run]"), msg)
}

// Output prints captured command output with a label.
func Output(label, output string) {
	if strings.TrimSpace(output) == "" {
		return
	}
	fmt.Printf("  %s:\n", c(ansiBold, label))
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		fmt.Printf("    %s\n", line)
	}
}

// RegisterNote prints a note that a result was registered, with its value.
func RegisterNote(varName, value string) {
	if strings.TrimSpace(value) != "" {
		fmt.Printf("  %s => %s: %s\n", c(ansiBlue, "registered"), varName, strings.TrimSpace(value))
	} else {
		fmt.Printf("  %s => %s\n", c(ansiBlue, "registered"), varName)
	}
}

// Recap prints the final PLAY RECAP table.
func Recap(summaries []HostSummary) {
	fmt.Printf("\n%s%s\n", c(ansiBold, "PLAY RECAP "), strings.Repeat("*", 62))
	for _, s := range summaries {
		hostStr := pad(s.Host, 24)
		if s.Failed > 0 {
			hostStr = c(ansiRed, hostStr)
		} else if s.Changed > 0 {
			hostStr = c(ansiYellow, hostStr)
		} else {
			hostStr = c(ansiGreen, hostStr)
		}
		ok := c(ansiGreen, fmt.Sprintf("ok=%-4d", s.OK))
		chg := c(ansiYellow, fmt.Sprintf("changed=%-4d", s.Changed))
		fail := c(ansiRed, fmt.Sprintf("failed=%-4d", s.Failed))
		skip := c(ansiCyan, fmt.Sprintf("skipped=%-4d", s.Skipped))
		ign := c(ansiYellow, fmt.Sprintf("ignored=%-4d", s.Ignored))
		fmt.Printf("  %s : %s %s %s %s %s\n", hostStr, ok, chg, fail, skip, ign)
	}
	fmt.Println()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
