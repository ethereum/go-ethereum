package utils

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
)

// Custom type which is registered in the flags library which cli uses for
// argument parsing. This allows us to expand Value to an absolute path when
// the argument is parsed
type DirectoryString struct {
	Value string
}

func (self *DirectoryString) String() string {
	return self.Value
}

func (self *DirectoryString) Set(value string) error {
	self.Value = expandPath(value)
	return nil
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	cli.GenericFlag
	Name   string
	Value  DirectoryString
	Usage  string
	EnvVar string
}

func (self DirectoryFlag) String() string {
	var fmtString string
	fmtString = "%s %v\t%v"

	if len(self.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	} else {
		fmtString = "%s %v\t%v"
	}

	return withEnvHint(self.EnvVar, fmt.Sprintf(fmtString, prefixedNames(self.Name), self.Value.Value, self.Usage))
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

// called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (self DirectoryFlag) Apply(set *flag.FlagSet) {
	if self.EnvVar != "" {
		for _, envVar := range strings.Split(self.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal := os.Getenv(envVar); envVal != "" {
				self.Value.Value = envVal
				break
			}
		}
	}

	eachName(self.Name, func(name string) {
		set.Var(&self.Value, self.Name, self.Usage)
	})
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return
}

func withEnvHint(envVar, str string) string {
	envText := ""
	if envVar != "" {
		envText = fmt.Sprintf(" [$%s]", strings.Join(strings.Split(envVar, ","), ", $"))
	}
	return str + envText
}

func (self DirectoryFlag) getName() string {
	return self.Name
}

func (self *DirectoryFlag) Set(value string) {
	self.Value.Value = value
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if user, err := user.Current(); err == nil {
			if err == nil {
				p = strings.Replace(p, "~", user.HomeDir, 1)
			}
		}
	}

	return filepath.Clean(os.ExpandEnv(p))
}
