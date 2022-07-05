package flags

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
	"github.com/urfave/cli/v2"
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
	Name string

	Category    string
	DefaultText string
	Usage       string

	Required   bool
	Hidden     bool
	HasBeenSet bool

	Value DirectoryString

	Aliases []string
}

// For cli.Flag:

func (f *DirectoryFlag) Names() []string { return append([]string{f.Name}, f.Aliases...) }
func (f *DirectoryFlag) IsSet() bool     { return f.HasBeenSet }
func (f *DirectoryFlag) String() string  { return cli.FlagStringer(f) }

// Apply called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (f *DirectoryFlag) Apply(set *flag.FlagSet) error {
	eachName(f, func(name string) {
		set.Var(&f.Value, f.Name, f.Usage)
	})
	return nil
}

// For cli.RequiredFlag:

func (f *DirectoryFlag) IsRequired() bool { return f.Required }

// For cli.VisibleFlag:

func (f *DirectoryFlag) IsVisible() bool { return !f.Hidden }

// For cli.CategorizableFlag:

func (f *DirectoryFlag) GetCategory() string { return f.Category }

// For cli.DocGenerationFlag:

func (f *DirectoryFlag) TakesValue() bool     { return true }
func (f *DirectoryFlag) GetUsage() string     { return f.Usage }
func (f *DirectoryFlag) GetValue() string     { return f.Value.String() }
func (f *DirectoryFlag) GetEnvVars() []string { return nil } // env not supported

func (f *DirectoryFlag) GetDefaultText() string {
	if f.DefaultText != "" {
		return f.DefaultText
	}
	return f.GetValue()
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := HomeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func HomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
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
	Name string

	Category    string
	DefaultText string
	Usage       string

	Required   bool
	Hidden     bool
	HasBeenSet bool

	Value TextMarshaler

	Aliases []string
}

// For cli.Flag:

func (f *TextMarshalerFlag) Names() []string { return append([]string{f.Name}, f.Aliases...) }
func (f *TextMarshalerFlag) IsSet() bool     { return f.HasBeenSet }
func (f *TextMarshalerFlag) String() string  { return cli.FlagStringer(f) }

func (f *TextMarshalerFlag) Apply(set *flag.FlagSet) error {
	eachName(f, func(name string) {
		set.Var(textMarshalerVal{f.Value}, f.Name, f.Usage)
	})
	return nil
}

// For cli.RequiredFlag:

func (f *TextMarshalerFlag) IsRequired() bool { return f.Required }

// For cli.VisibleFlag:

func (f *TextMarshalerFlag) IsVisible() bool { return !f.Hidden }

// For cli.CategorizableFlag:

func (f *TextMarshalerFlag) GetCategory() string { return f.Category }

// For cli.DocGenerationFlag:

func (f *TextMarshalerFlag) TakesValue() bool     { return true }
func (f *TextMarshalerFlag) GetUsage() string     { return f.Usage }
func (f *TextMarshalerFlag) GetEnvVars() []string { return nil } // env not supported

func (f *TextMarshalerFlag) GetValue() string {
	t, err := f.Value.MarshalText()
	if err != nil {
		return "(ERR: " + err.Error() + ")"
	}
	return string(t)
}

func (f *TextMarshalerFlag) GetDefaultText() string {
	if f.DefaultText != "" {
		return f.DefaultText
	}
	return f.GetValue()
}

// GlobalTextMarshaler returns the value of a TextMarshalerFlag from the global flag set.
func GlobalTextMarshaler(ctx *cli.Context, name string) TextMarshaler {
	val := ctx.Generic(name)
	if val == nil {
		return nil
	}
	return val.(textMarshalerVal).v
}

// BigFlag is a command line flag that accepts 256 bit big integers in decimal or
// hexadecimal syntax.
type BigFlag struct {
	Name string

	Category    string
	DefaultText string
	Usage       string

	Required   bool
	Hidden     bool
	HasBeenSet bool

	Value *big.Int

	Aliases []string
}

// For cli.Flag:

func (f *BigFlag) Names() []string { return append([]string{f.Name}, f.Aliases...) }
func (f *BigFlag) IsSet() bool     { return f.HasBeenSet }
func (f *BigFlag) String() string  { return cli.FlagStringer(f) }

func (f *BigFlag) Apply(set *flag.FlagSet) error {
	eachName(f, func(name string) {
		f.Value = new(big.Int)
		set.Var((*bigValue)(f.Value), f.Name, f.Usage)
	})

	return nil
}

// For cli.RequiredFlag:

func (f *BigFlag) IsRequired() bool { return f.Required }

// For cli.VisibleFlag:

func (f *BigFlag) IsVisible() bool { return !f.Hidden }

// For cli.CategorizableFlag:

func (f *BigFlag) GetCategory() string { return f.Category }

// For cli.DocGenerationFlag:

func (f *BigFlag) TakesValue() bool     { return true }
func (f *BigFlag) GetUsage() string     { return f.Usage }
func (f *BigFlag) GetValue() string     { return f.Value.String() }
func (f *BigFlag) GetEnvVars() []string { return nil } // env not supported

func (f *BigFlag) GetDefaultText() string {
	if f.DefaultText != "" {
		return f.DefaultText
	}
	return f.GetValue()
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
	intVal, ok := math.ParseBig256(s)
	if !ok {
		return errors.New("invalid integer syntax")
	}
	*b = (bigValue)(*intVal)
	return nil
}

// GlobalBig returns the value of a BigFlag from the global flag set.
func GlobalBig(ctx *cli.Context, name string) *big.Int {
	val := ctx.Generic(name)
	if val == nil {
		return nil
	}
	return (*big.Int)(val.(*bigValue))
}

func eachName(f cli.Flag, fn func(string)) {
	for _, name := range f.Names() {
		name = strings.Trim(name, " ")
		fn(name)
	}
}
