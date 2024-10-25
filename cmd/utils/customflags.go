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
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/common/math"
	"gopkg.in/urfave/cli.v1"
)

// Custom type which is registered in the flags library which cli uses for
// argument parsing. This allows us to expand Value to an absolute path when
// the argument is parsed
type DirectoryString struct {
	Value string
}

func (ds *DirectoryString) String() string {
	return ds.Value
}

func (ds *DirectoryString) Set(value string) error {
	ds.Value = expandPath(value)
	return nil
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	Name  string
	Value DirectoryString
	Usage string
}

func (df DirectoryFlag) String() string {
	fmtString := "%s %v\t%v"
	if len(df.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(df.Name), df.Value.Value, df.Usage)
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
func (df DirectoryFlag) Apply(set *flag.FlagSet) {
	eachName(df.Name, func(name string) {
		set.Var(&df.Value, df.Name, df.Usage)
	})
}

type TextMarshaler interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// textMarshalerVal turns a TextMarshaler into a flag.Value
type textMarshalerVal struct {
	v TextMarshaler
}

func (t textMarshalerVal) String() string {
	if t.v == nil {
		return ""
	}
	text, _ := t.v.MarshalText()
	return string(text)
}

func (t textMarshalerVal) Set(s string) error {
	return t.v.UnmarshalText([]byte(s))
}

// TextMarshalerFlag wraps a TextMarshaler value.
type TextMarshalerFlag struct {
	Name  string
	Value TextMarshaler
	Usage string
}

func (t TextMarshalerFlag) GetName() string {
	return t.Name
}

func (t TextMarshalerFlag) String() string {
	return fmt.Sprintf("%s \"%v\"\t%v", prefixedNames(t.Name), t.Value, t.Usage)
}

func (t TextMarshalerFlag) Apply(set *flag.FlagSet) {
	eachName(t.Name, func(name string) {
		set.Var(textMarshalerVal{t.Value}, t.Name, t.Usage)
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
	Name  string
	Value *big.Int
	Usage string
}

// bigValue turns *big.Int into a flag.Value
type bigValue big.Int

func (bv *bigValue) String() string {
	if bv == nil {
		return ""
	}
	return (*big.Int)(bv).String()
}

func (bv *bigValue) Set(s string) error {
	int, ok := math.ParseBig256(s)
	if !ok {
		return errors.New("invalid integer syntax")
	}
	*bv = (bigValue)(*int)
	return nil
}

func (bf BigFlag) GetName() string {
	return bf.Name
}

func (bf BigFlag) String() string {
	fmtString := "%s %v\t%v"
	if bf.Value != nil {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(bf.Name), bf.Value, bf.Usage)
}

func (bf BigFlag) Apply(set *flag.FlagSet) {
	eachName(bf.Name, func(name string) {
		set.Var((*bigValue)(bf.Value), bf.Name, bf.Usage)
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

func (df DirectoryFlag) GetName() string {
	return df.Name
}

func (df *DirectoryFlag) Set(value string) {
	df.Value.Value = value
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
