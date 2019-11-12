// Copyright 2015 The go-ethereum Authors
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

package utils

import (
	"encoding"
	"errors"
	"flag"
	"math/big"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/common/math"
	"gopkg.in/urfave/cli.v1"
)

// Custom type which is registered in the flags library which cli uses for
// argument parsing. This allows us to expand Value to an absolute path when
// the argument is parsed
type DirectoryString string

func (s *DirectoryString) String() string {
	return string(*s)
}

func (s *DirectoryString) Set(value string) error {
	*s = DirectoryString(expandPath(value))
	return nil
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	Name   string
	Value  DirectoryString
	Usage  string
	EnvVar string
}

func (f DirectoryFlag) String() string {
	return cli.FlagStringer(f)
}

// called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (f DirectoryFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var(&f.Value, f.Name, f.Usage)
	})
}

func (f DirectoryFlag) GetName() string {
	return f.Name
}

func (f *DirectoryFlag) Set(value string) {
	f.Value.Set(value)
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

type TextMarshaler interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// textMarshalerVal turns a TextMarshaler into a flag.Value
type textMarshalerVal struct {
	v TextMarshaler
}

func (v textMarshalerVal) String() string {
	if v.v == nil {
		return ""
	}
	text, _ := v.v.MarshalText()
	return string(text)
}

func (v textMarshalerVal) Set(s string) error {
	return v.v.UnmarshalText([]byte(s))
}

// TextMarshalerFlag wraps a TextMarshaler value.
type TextMarshalerFlag struct {
	Name   string
	Value  TextMarshaler
	Usage  string
	EnvVar string
}

func (f TextMarshalerFlag) GetName() string {
	return f.Name
}

func (f TextMarshalerFlag) String() string {
	return cli.FlagStringer(f)
}

func (f TextMarshalerFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var(textMarshalerVal{f.Value}, f.Name, f.Usage)
	})
}

// GlobalTextMarshaler returns the value of a TextMarshalerFlag from the global flag set.
func GlobalTextMarshaler(ctx *cli.Context, name string) TextMarshaler {
	val := ctx.GlobalGeneric(name)
	if val == nil {
		return nil
	}
	return val.(textMarshalerVal).v
}

// BigFlag is a command line flag that accepts 256 bit big integers in decimal or
// hexadecimal syntax.
type BigFlag struct {
	Name   string
	Value  *big.Int
	Usage  string
	EnvVar string
}

// bigValue turns *big.Int into a flag.Value
type bigValue big.Int

func (b *bigValue) String() string {
	if b == nil {
		return ""
	}
	return (*big.Int)(b).String()
}

func (b *bigValue) Set(s string) error {
	int, ok := math.ParseBig256(s)
	if !ok {
		return errors.New("invalid integer syntax")
	}
	*b = (bigValue)(*int)
	return nil
}

func (f BigFlag) GetName() string {
	return f.Name
}

func (f BigFlag) String() string {
	return cli.FlagStringer(f)
}

func (f BigFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var((*bigValue)(f.Value), f.Name, f.Usage)
	})
}

// GlobalBig returns the value of a BigFlag from the global flag set.
func GlobalBig(ctx *cli.Context, name string) *big.Int {
	val := ctx.GlobalGeneric(name)
	if val == nil {
		return nil
	}
	return (*big.Int)(val.(*bigValue))
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

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
