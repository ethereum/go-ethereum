package flagset

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Flagset struct {
	flags []*FlagVar
	set   *flag.FlagSet
}

func NewFlagSet(name string) *Flagset {
	f := &Flagset{
		flags: []*FlagVar{},
		set:   flag.NewFlagSet(name, flag.ContinueOnError),
	}
	return f
}

type FlagVar struct {
	Name  string
	Usage string
}

func (f *Flagset) addFlag(fl *FlagVar) {
	f.flags = append(f.flags, fl)
}

func (f *Flagset) Help() string {
	str := "Options:\n\n"
	items := []string{}
	for _, item := range f.flags {
		items = append(items, fmt.Sprintf("  -%s\n    %s", item.Name, item.Usage))
	}
	return str + strings.Join(items, "\n\n")
}

func (f *Flagset) Parse(args []string) error {
	return f.set.Parse(args)
}

func (f *Flagset) Args() []string {
	return f.set.Args()
}

func (f *Flagset) BoolVar() {

}

type BoolFlag struct {
	Name    string
	Usage   string
	Default bool
	Value   *bool
}

func (f *Flagset) BoolFlag(b *BoolFlag) {
	f.addFlag(&FlagVar{
		Name:  b.Name,
		Usage: b.Usage,
	})
	f.set.BoolVar(b.Value, b.Name, b.Default, b.Usage)
}

type StringFlag struct {
	Name    string
	Usage   string
	Default string
	Value   *string
}

func (f *Flagset) StringFlag(b *StringFlag) {
	f.addFlag(&FlagVar{
		Name:  b.Name,
		Usage: b.Usage,
	})
	f.set.StringVar(b.Value, b.Name, b.Default, b.Usage)
}

type IntFlag struct {
	Name    string
	Usage   string
	Default int
	Value   *int
}

func (f *Flagset) IntFlag(i *IntFlag) {
	f.addFlag(&FlagVar{
		Name:  i.Name,
		Usage: i.Usage,
	})
	f.set.IntVar(i.Value, i.Name, i.Default, i.Usage)
}

type SliceStringFlag struct {
	Name  string
	Usage string
	Value []string
}

func (i *SliceStringFlag) String() string {
	return ""
}

func (i *SliceStringFlag) Set(value string) error {
	i.Value = append(i.Value, value)
	return nil
}

func (f *Flagset) SliceStringFlag(s *SliceStringFlag) {
	f.addFlag(&FlagVar{
		Name:  s.Name,
		Usage: s.Usage,
	})
	f.set.Var(s, s.Name, s.Usage)
}

type DurationFlag struct {
	Name  string
	Usage string
	Value *time.Duration
}

func (f *Flagset) DurationFlag(d *DurationFlag) {
	f.addFlag(&FlagVar{
		Name:  d.Name,
		Usage: d.Usage,
	})
}
