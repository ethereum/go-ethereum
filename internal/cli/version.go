package cli

import (
	"strings"

	"github.com/ethereum/go-ethereum/params"

	"github.com/mitchellh/cli"
)

// VersionCommand is the command to show the version of the agent
type VersionCommand struct {
	UI cli.Ui
}

// MarkDown implements cli.MarkDown interface
func (d *VersionCommand) MarkDown() string {
	examples := []string{
		"## Usage",
		CodeBlock([]string{
			"$ bor version",
			"0.2.9-stable",
		}),
	}

	items := []string{
		"# Version",
		"The ```bor version``` command outputs the version of the binary.",
	}
	items = append(items, examples...)

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *VersionCommand) Help() string {
	return `Usage: bor version

  Display the Bor version`
}

// Synopsis implements the cli.Command interface
func (c *VersionCommand) Synopsis() string {
	return "Display the Bor version"
}

// Run implements the cli.Command interface
func (c *VersionCommand) Run(args []string) int {
	c.UI.Output(params.VersionWithMetaCommitDetails)

	return 0
}
