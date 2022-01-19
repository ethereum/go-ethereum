package flagset

import (
	"flag"
	"fmt"
	"math/big"
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
	Value   *int
	Default int
}

func (f *Flagset) IntFlag(i *IntFlag) {
	f.addFlag(&FlagVar{
		Name:  i.Name,
		Usage: i.Usage,
	})
	f.set.IntVar(i.Value, i.Name, i.Default, i.Usage)
}

type Uint64Flag struct {
	Name    string
	Usage   string
	Value   *uint64
	Default uint64
}

func (f *Flagset) Uint64Flag(i *Uint64Flag) {
	f.addFlag(&FlagVar{
		Name:  i.Name,
		Usage: i.Usage,
	})
	f.set.Uint64Var(i.Value, i.Name, i.Default, i.Usage)
}

type BigIntFlag struct {
	Name  string
	Usage string
	Value *big.Int
}

func (b *BigIntFlag) String() string {
	if b.Value == nil {
		return ""
	}
	return b.Value.String()
}

func (b *BigIntFlag) Set(value string) error {
	num := new(big.Int)

	var ok bool
	if strings.HasPrefix(value, "0x") {
		num, ok = num.SetString(value[2:], 16)
	} else {
		num, ok = num.SetString(value, 10)
	}
	if !ok {
		return fmt.Errorf("failed to set big int")
	}
	b.Value = num
	return nil
}

func (f *Flagset) BigIntFlag(b *BigIntFlag) {
	f.addFlag(&FlagVar{
		Name:  b.Name,
		Usage: b.Usage,
	})
	f.set.Var(b, b.Name, b.Usage)
}

type SliceStringFlag struct {
	Name  string
	Usage string
	Value *[]string
}

func (i *SliceStringFlag) String() string {
	if i.Value == nil {
		return ""
	}
	return strings.Join(*i.Value, ",")
}

func (i *SliceStringFlag) Set(value string) error {
	*i.Value = append(*i.Value, strings.Split(value, ",")...)
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
	Name    string
	Usage   string
	Value   *time.Duration
	Default time.Duration
}

func (f *Flagset) DurationFlag(d *DurationFlag) {
	f.addFlag(&FlagVar{
		Name:  d.Name,
		Usage: d.Usage,
	})
	f.set.DurationVar(d.Value, d.Name, d.Default, "")
}

type MapStringFlag struct {
	Name  string
	Usage string
	Value *map[string]string
}

func (m *MapStringFlag) String() string {
	if m.Value == nil {
		return ""
	}
	ls := []string{}
	for k, v := range *m.Value {
		ls = append(ls, k+"="+v)
	}
	return strings.Join(ls, ",")
}

func (m *MapStringFlag) Set(value string) error {
	if m.Value == nil {
		m.Value = &map[string]string{}
	}
	for _, t := range strings.Split(value, ",") {
		if t != "" {
			kv := strings.Split(t, "=")

			if len(kv) == 2 {
				(*m.Value)[kv[0]] = kv[1]
			}
		}
	}
	return nil
}

func (f *Flagset) MapStringFlag(m *MapStringFlag) {
	f.addFlag(&FlagVar{
		Name:  m.Name,
		Usage: m.Usage,
	})
	f.set.Var(m, m.Name, m.Usage)
}

type Float64Flag struct {
	Name    string
	Usage   string
	Value   *float64
	Default float64
}

func (f *Flagset) Float64Flag(i *Float64Flag) {
	f.addFlag(&FlagVar{
		Name:  i.Name,
		Usage: i.Usage,
	})
	f.set.Float64Var(i.Value, i.Name, i.Default, "")
}
