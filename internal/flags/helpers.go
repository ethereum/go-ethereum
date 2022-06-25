// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package flags

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
)

func init() {
	cli.FlagStringer = FlagString
}

// NewApp creates an app with sane defaults.
func NewApp(gitCommit, gitDate, usage string) *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	app.Copyright = "Copyright 2013-2022 The go-ethereum Authors"
	return app
}

// FlagString prints a single flag in help.
func FlagString(f cli.Flag) string {
	df, ok := f.(cli.DocGenerationFlag)
	if !ok {
		return ""
	}

	needsPlaceholder := df.TakesValue()
	placeholder := ""
	if needsPlaceholder {
		placeholder = "value"
	}

	namesText := pad(cli.FlagNamePrefixer(df.Names(), placeholder), 30)

	defaultValueString := ""
	if s := df.GetDefaultText(); s != "" {
		defaultValueString = " (default: " + s + ")"
	}

	usage := strings.TrimSpace(df.GetUsage())
	envHint := strings.TrimSpace(cli.FlagEnvHinter(df.GetEnvVars(), ""))
	if len(envHint) > 0 {
		usage += " " + envHint
	}

	usage = wordWrap(usage, 80)
	usage = indent(usage, 10)

	return fmt.Sprintf("\n    %s%s\n%s", namesText, defaultValueString, usage)
}

func pad(s string, length int) string {
	if len(s) < length {
		s += strings.Repeat(" ", length-len(s))
	}
	return s
}

func indent(s string, nspace int) string {
	ind := strings.Repeat(" ", nspace)
	return ind + strings.ReplaceAll(s, "\n", "\n"+ind)
}

func wordWrap(s string, width int) string {
	var (
		output     strings.Builder
		lineLength = 0
	)

	for {
		sp := strings.IndexByte(s, ' ')
		var word string
		if sp == -1 {
			word = s
		} else {
			word = s[:sp]
		}
		wlen := len(word)
		over := lineLength+wlen >= width
		if over {
			output.WriteByte('\n')
			lineLength = 0
		} else {
			if lineLength != 0 {
				output.WriteByte(' ')
				lineLength++
			}
		}

		output.WriteString(word)
		lineLength += wlen

		if sp == -1 {
			break
		}
		s = s[wlen+1:]
	}

	return output.String()
}
