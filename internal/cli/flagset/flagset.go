package flagset

import (
	"flag"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Flagset struct {
	flags map[string]*FlagVar
	set   *flag.FlagSet
}

func NewFlagSet(name string) *Flagset {
	f := &Flagset{
		flags: make(map[string]*FlagVar, 0),
		set:   flag.NewFlagSet(name, flag.ContinueOnError),
	}

	return f
}

// Updatable is a minimalistic representation of a flag which has
// the method `UpdateValue` implemented which can be called while
// overwriting flags.
type Updatable interface {
	UpdateValue(string)
}

type FlagVar struct {
	Name    string
	Usage   string
	Group   string
	Default any
	Value   Updatable
}

// ByName implements sort.Interface for []*FlagVar based on the Name field.
type ByName []*FlagVar

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (f *Flagset) addFlag(fl *FlagVar) {
	f.flags[fl.Name] = fl
}

func (f *Flagset) Help() string {
	str := "Options:\n\n"

	items := []string{}

	flags := []*FlagVar{}
	for _, item := range f.flags {
		flags = append(flags, item)
	}

	sort.Sort(ByName(flags))

	for _, item := range flags {
		if item.Default != nil {
			items = append(items, fmt.Sprintf("  -%s\n    %s (default: %v)", item.Name, item.Usage, item.Default))
		} else {
			items = append(items, fmt.Sprintf("  -%s\n    %s", item.Name, item.Usage))
		}
	}

	return str + strings.Join(items, "\n\n")
}

func (f *Flagset) GetAllFlags() []string {
	i := 0
	flags := make([]string, 0, len(f.flags))

	for name := range f.flags {
		flags[i] = name
		i++
	}

	return flags
}

// MarkDown implements cli.MarkDown interface
func (f *Flagset) MarkDown() string {
	if len(f.flags) == 0 {
		return ""
	}

	groups := make(map[string][]*FlagVar)

	for _, item := range f.flags {
		groups[item.Group] = append(groups[item.Group], item)
	}

	i := 0
	keys := make([]string, len(groups))

	for k := range groups {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	items := []string{}

	for _, k := range keys {
		if k == "" {
			items = append(items, "## Options")
		} else {
			items = append(items, fmt.Sprintf("### %s Options", k))
		}

		flags := make([]*FlagVar, len(groups[k]))
		copy(flags, groups[k])
		sort.Sort(ByName(flags))

		for _, item := range flags {
			if item.Default != nil {
				items = append(items, fmt.Sprintf("- ```%s```: %s (default: %v)", item.Name, item.Usage, item.Default))
			} else {
				items = append(items, fmt.Sprintf("- ```%s```: %s", item.Name, item.Usage))
			}
		}
	}

	return strings.Join(items, "\n\n")
}

func (f *Flagset) Parse(args []string) error {
	return f.set.Parse(args)
}

func (f *Flagset) Args() []string {
	return f.set.Args()
}

// UpdateValue updates the underlying value of a flag
// given the flag name and value to update using pointer.
func (f *Flagset) UpdateValue(names []string, values []string) {
	for i, name := range names {
		if flag, ok := f.flags[name]; ok {
			value := values[i]

			// Call the underlying flag's `UpdateValue` method
			flag.Value.UpdateValue(value)
		}
	}
}

// Visit visits all the set flags and returns the name and value
// in string to set later.
func (f *Flagset) Visit() ([]string, []string) {
	names := make([]string, 0, len(f.flags))
	values := make([]string, 0, len(f.flags))

	f.set.Visit(func(flag *flag.Flag) {
		names = append(names, flag.Name)
		values = append(values, flag.Value.String())
	})

	return names, values
}

type BoolFlag struct {
	Name    string
	Usage   string
	Default bool
	Value   *bool
	Group   string
}

func (b *BoolFlag) UpdateValue(value string) {
	v, _ := strconv.ParseBool(value)

	*b.Value = v
}

func (f *Flagset) BoolFlag(b *BoolFlag) {
	f.addFlag(&FlagVar{
		Name:    b.Name,
		Usage:   b.Usage,
		Group:   b.Group,
		Default: b.Default,
		Value:   b,
	})
	f.set.BoolVar(b.Value, b.Name, b.Default, b.Usage)
}

type StringFlag struct {
	Name               string
	Usage              string
	Default            string
	Value              *string
	Group              string
	HideDefaultFromDoc bool
}

func (b *StringFlag) UpdateValue(value string) {
	*b.Value = value
}

func (f *Flagset) StringFlag(b *StringFlag) {
	if b.Default == "" || b.HideDefaultFromDoc {
		f.addFlag(&FlagVar{
			Name:    b.Name,
			Usage:   b.Usage,
			Group:   b.Group,
			Default: nil,
			Value:   b,
		})
	} else {
		f.addFlag(&FlagVar{
			Name:    b.Name,
			Usage:   b.Usage,
			Group:   b.Group,
			Default: b.Default,
			Value:   b,
		})
	}

	f.set.StringVar(b.Value, b.Name, b.Default, b.Usage)
}

type IntFlag struct {
	Name    string
	Usage   string
	Value   *int
	Default int
	Group   string
}

func (b *IntFlag) UpdateValue(value string) {
	v, _ := strconv.ParseInt(value, 10, 64)

	*b.Value = int(v)
}

func (f *Flagset) IntFlag(i *IntFlag) {
	f.addFlag(&FlagVar{
		Name:    i.Name,
		Usage:   i.Usage,
		Group:   i.Group,
		Default: i.Default,
		Value:   i,
	})
	f.set.IntVar(i.Value, i.Name, i.Default, i.Usage)
}

type Uint64Flag struct {
	Name    string
	Usage   string
	Value   *uint64
	Default uint64
	Group   string
}

func (b *Uint64Flag) UpdateValue(value string) {
	v, _ := strconv.ParseUint(value, 10, 64)

	*b.Value = v
}

func (f *Flagset) Uint64Flag(i *Uint64Flag) {
	f.addFlag(&FlagVar{
		Name:    i.Name,
		Usage:   i.Usage,
		Group:   i.Group,
		Default: fmt.Sprintf("%d", i.Default),
		Value:   i,
	})
	f.set.Uint64Var(i.Value, i.Name, i.Default, i.Usage)
}

type BigIntFlag struct {
	Name    string
	Usage   string
	Value   *big.Int
	Group   string
	Default *big.Int
}

func (b *BigIntFlag) String() string {
	if b.Value == nil {
		return ""
	}

	return b.Value.String()
}

func parseBigInt(value string) *big.Int {
	num := new(big.Int)

	if strings.HasPrefix(value, "0x") {
		num, _ = num.SetString(value[2:], 16)
	} else {
		num, _ = num.SetString(value, 10)
	}

	return num
}

func (b *BigIntFlag) Set(value string) error {
	num := parseBigInt(value)

	if num == nil {
		return fmt.Errorf("failed to set big int")
	}

	*b.Value = *num

	return nil
}

func (b *BigIntFlag) UpdateValue(value string) {
	num := parseBigInt(value)

	if num == nil {
		return
	}

	*b.Value = *num
}

func (f *Flagset) BigIntFlag(b *BigIntFlag) {
	f.addFlag(&FlagVar{
		Name:    b.Name,
		Usage:   b.Usage,
		Group:   b.Group,
		Default: b.Default,
		Value:   b,
	})
	f.set.Var(b, b.Name, b.Usage)
}

type SliceStringFlag struct {
	Name    string
	Usage   string
	Value   *[]string
	Default []string
	Group   string
}

// SplitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func SplitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		if r = strings.TrimSpace(r); r != "" {
			ret = append(ret, r)
		}
	}

	return ret
}

func (i *SliceStringFlag) String() string {
	if i.Value == nil {
		return ""
	}

	return strings.Join(*i.Value, ",")
}

func (i *SliceStringFlag) Set(value string) error {
	// overwriting instead of appending
	*i.Value = SplitAndTrim(value)
	return nil
}

func (i *SliceStringFlag) UpdateValue(value string) {
	*i.Value = SplitAndTrim(value)
}

func (f *Flagset) SliceStringFlag(s *SliceStringFlag) {
	if s.Default == nil || len(s.Default) == 0 {
		f.addFlag(&FlagVar{
			Name:    s.Name,
			Usage:   s.Usage,
			Group:   s.Group,
			Default: nil,
			Value:   s,
		})
	} else {
		f.addFlag(&FlagVar{
			Name:    s.Name,
			Usage:   s.Usage,
			Group:   s.Group,
			Default: strings.Join(s.Default, ","),
			Value:   s,
		})
	}

	f.set.Var(s, s.Name, s.Usage)
}

type DurationFlag struct {
	Name    string
	Usage   string
	Value   *time.Duration
	Default time.Duration
	Group   string
}

func (d *DurationFlag) UpdateValue(value string) {
	v, _ := time.ParseDuration(value)

	*d.Value = v
}

func (f *Flagset) DurationFlag(d *DurationFlag) {
	f.addFlag(&FlagVar{
		Name:    d.Name,
		Usage:   d.Usage,
		Group:   d.Group,
		Default: d.Default,
		Value:   d,
	})
	f.set.DurationVar(d.Value, d.Name, d.Default, "")
}

type MapStringFlag struct {
	Name    string
	Usage   string
	Value   *map[string]string
	Group   string
	Default map[string]string
}

func formatMapString(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}

	ls := []string{}
	for k, v := range m {
		ls = append(ls, k+"="+v)
	}

	return strings.Join(ls, ",")
}

func (m *MapStringFlag) String() string {
	return formatMapString(*m.Value)
}

func parseMap(value string) map[string]string {
	m := make(map[string]string)

	for _, t := range strings.Split(value, ",") {
		if t != "" {
			kv := strings.Split(t, "=")

			if len(kv) == 2 {
				m[kv[0]] = kv[1]
			}
		}
	}

	return m
}

func (m *MapStringFlag) Set(value string) error {
	if m.Value == nil {
		m.Value = &map[string]string{}
	}

	m2 := parseMap(value)
	*m.Value = m2

	return nil
}

func (m *MapStringFlag) UpdateValue(value string) {
	m2 := parseMap(value)
	*m.Value = m2
}

func (f *Flagset) MapStringFlag(m *MapStringFlag) {
	if m.Default == nil || len(m.Default) == 0 {
		f.addFlag(&FlagVar{
			Name:    m.Name,
			Usage:   m.Usage,
			Group:   m.Group,
			Default: nil,
			Value:   m,
		})
	} else {
		f.addFlag(&FlagVar{
			Name:    m.Name,
			Usage:   m.Usage,
			Group:   m.Group,
			Default: formatMapString(m.Default),
			Value:   m,
		})
	}

	f.set.Var(m, m.Name, m.Usage)
}

type Float64Flag struct {
	Name    string
	Usage   string
	Value   *float64
	Default float64
	Group   string
}

func (f *Float64Flag) UpdateValue(value string) {
	v, _ := strconv.ParseFloat(value, 64)

	*f.Value = v
}

func (f *Flagset) Float64Flag(i *Float64Flag) {
	f.addFlag(&FlagVar{
		Name:    i.Name,
		Usage:   i.Usage,
		Group:   i.Group,
		Default: i.Default,
		Value:   i,
	})
	f.set.Float64Var(i.Value, i.Name, i.Default, "")
}
