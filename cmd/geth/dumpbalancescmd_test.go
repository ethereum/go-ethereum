package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

// TestDumpBalancesCommandRegistered checks that our dump-balances
// command is present in app.Commands with the correct Usage text.
func TestDumpBalancesCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range app.Commands {
		if cmd.Name == dumpBalancesCommand.Name {
			found = true
			// Ensure that the Usage string mentions "non-zero accounts"
			if !strings.Contains(cmd.Usage, "non-zero accounts") {
				t.Errorf("Usage for %q does not include expected text, got %q", cmd.Name, cmd.Usage)
			}
			break
		}
	}
	if !found {
		t.Fatalf("command %q is not registered in app.Commands", dumpBalancesCommand.Name)
	}
}

// TestDumpBalancesHelpInProcess simulates running `geth dump-balances --help`
// in memory and verifies that help output includes the command name and its Usage.
func TestDumpBalancesHelpInProcess(t *testing.T) {
	// Create a fresh CLI app with our commands
	app := cli.NewApp()
	app.Commands = append([]*cli.Command{}, app.Commands...) // copy existing commands
	// No need to re-register dumpBalancesCommand because init() already ran

	// Capture the help output in a buffer
	buf := &bytes.Buffer{}
	app.Writer = buf

	// Run the help command; cli returns flag.ErrHelp on --help
	err := app.Run([]string{"geth", "dump-balances", "--help"})
	if err == nil {
		t.Fatalf("expected error when running help, got nil")
	}

	output := buf.String()
	if !strings.Contains(output, "dump-balances") {
		t.Errorf("help output missing command name; got:\n%s", output)
	}
	if !strings.Contains(output, "non-zero accounts") {
		t.Errorf("help output missing Usage description; got:\n%s", output)
	}
}
