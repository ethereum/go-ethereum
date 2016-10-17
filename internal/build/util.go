// Copyright 2016 The go-ethereum Authors
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

package build

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var DryRunFlag = flag.Bool("n", false, "dry run, don't execute commands")

// MustRun executes the given command and exits the host process for
// any error.
func MustRun(cmd *exec.Cmd) {
	fmt.Println(">>>", strings.Join(cmd.Args, " "))
	if !*DryRunFlag {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

func MustRunCommand(cmd string, args ...string) {
	MustRun(exec.Command(cmd, args...))
}

// GOPATH returns the value that the GOPATH environment
// variable should be set to.
func GOPATH() string {
	path := filepath.SplitList(os.Getenv("GOPATH"))
	if len(path) == 0 {
		log.Fatal("GOPATH is not set")
	}
	// Ensure that our internal vendor folder is on GOPATH
	vendor, _ := filepath.Abs(filepath.Join("build", "_vendor"))
	for _, dir := range path {
		if dir == vendor {
			return strings.Join(path, string(filepath.ListSeparator))
		}
	}
	newpath := append(path[:1], append([]string{vendor}, path[1:]...)...)
	return strings.Join(newpath, string(filepath.ListSeparator))
}

// VERSION returns the content of the VERSION file.
func VERSION() string {
	version, err := ioutil.ReadFile("VERSION")
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes.TrimSpace(version))
}

// RunGit runs a git subcommand and returns its output.
// The command must complete successfully.
func RunGit(args ...string) string {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err == exec.ErrNotFound {
		log.Println("no git in PATH")
		return ""
	} else if err != nil {
		log.Fatal(strings.Join(cmd.Args, " "), ": ", err, "\n", stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

// Render renders the given template file into outputFile.
func Render(templateFile, outputFile string, outputPerm os.FileMode, x interface{}) {
	tpl := template.Must(template.ParseFiles(templateFile))
	render(tpl, outputFile, outputPerm, x)
}

// RenderString renders the given template string into outputFile.
func RenderString(templateContent, outputFile string, outputPerm os.FileMode, x interface{}) {
	tpl := template.Must(template.New("").Parse(templateContent))
	render(tpl, outputFile, outputPerm, x)
}

func render(tpl *template.Template, outputFile string, outputPerm os.FileMode, x interface{}) {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		log.Fatal(err)
	}
	out, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, outputPerm)
	if err != nil {
		log.Fatal(err)
	}
	if err := tpl.Execute(out, x); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}

// CopyFile copies a file.
func CopyFile(dst, src string, mode os.FileMode) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		log.Fatal(err)
	}
	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		log.Fatal(err)
	}
	defer destFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		log.Fatal(err)
	}
}
