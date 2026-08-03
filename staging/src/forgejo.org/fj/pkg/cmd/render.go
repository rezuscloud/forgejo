package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Rendering helpers for colored CLI output (Tier 4 polish).

var (
	// Detection: respect NO_COLOR and non-TTY
	colorEnabled = !isNoColor() && isTerminal()

	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	grey   = color.New(color.FgHiBlack)
	bold   = color.New(color.Bold)
	cyan   = color.New(color.FgCyan)
)

func init() {
	if !colorEnabled {
		color.NoColor = true
	}
}

func isNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

func isTerminal() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// statusSymbol returns a colored status glyph for action/issue/PR states.
func statusSymbol(status string) string {
	if !colorEnabled {
		return plainStatusSymbol(status)
	}
	switch status {
	case "success", "open", "merged":
		return green.Sprint("✓")
	case "failure", "closed":
		return red.Sprint("✗")
	case "running":
		return yellow.Sprint("●")
	case "waiting", "queued", "blocked":
		return grey.Sprint("⋯")
	case "cancelled", "skipped":
		return grey.Sprint("⊘")
	default:
		return grey.Sprint("?")
	}
}

func plainStatusSymbol(status string) string {
	switch status {
	case "success", "open", "merged":
		return "OK"
	case "failure", "closed":
		return "FAIL"
	case "running":
		return "RUN"
	case "waiting", "queued", "blocked":
		return "WAIT"
	case "cancelled", "skipped":
		return "SKIP"
	default:
		return status
	}
}

// printErr prints an error message in red to stderr.
func printErr(msg string) {
	fmt.Fprintln(os.Stderr, red.Sprint("error: ")+msg)
}

// printOK prints a success message in green.
func printOK(msg string) {
	fmt.Println(green.Sprint(msg))
}

// suppress unused import warning
var _ = io.Discard
var _ = cyan
var _ = bold
