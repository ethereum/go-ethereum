// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"html/template"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/ethereum/go-ethereum/params"

	cli "gopkg.in/urfave/cli.v1"
)

var bugCommand = cli.Command{
	Action:    reportBug,
	Name:      "bug",
	Usage:     "opens a window to report a bug on the geth repo",
	ArgsUsage: " ",
	Category:  "MISCELLANEOUS COMMANDS",
}

// reportBug reports a bug by opening a new URL to the go-ethereum GH issue
// tracker and setting default values as the issue body.
func reportBug(ctx *cli.Context) error {
	// compile the bug report template or crash
	t := template.Must(template.New("bug-report").Parse(bugTemplate))

	// assemble the info for the bug template
	uname, err := uname()
	if err != nil {
		return err
	}

	info := struct {
		Version, GoVersion, OS, Uname string
	}{Version: params.Version, GoVersion: runtime.Version(), OS: runtime.GOOS, Uname: uname}

	// execute template and write contents to buff
	var buff bytes.Buffer
	if err := t.Execute(&buff, info); err != nil {
		return err
	}

	// open a new GH issue
	return exec.Command("open", "https://github.com/ethereum/go-ethereum/issues/new?body="+url.QueryEscape(buff.String())).Start()
}

const bugTemplate = `Please answer these questions before submitting your issue. Thanks!

#### What did you do?

#### What did you expect to see?

#### What did you see instead?

#### System details

Version: {{.Version}}

Go Version: {{.GoVersion}}
OS: {{.OS}}
uname -a: {{.Uname}}
`

// uname returns machine hardware, os release, name and version.
func uname() (string, error) {
	// figure out the uname of the system (e.g. uname, systeminfo)
	var (
		err   error
		uname []byte
		cmd   = [2]string{"uname", "-a"}
	)

	if runtime.GOOS == "windows" {
		cmd = [2]string{"systeminfo"}
	}

	uname, err = exec.Command(cmd[0], cmd[1]).Output()
	if err != nil {
		return "", err
	}
	return string(uname), err
}
