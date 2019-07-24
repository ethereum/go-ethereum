package pflag

import (
	"fmt"
	"strconv"
	"strings"
)

// -- int32Slice Value
type int32SliceValue struct {
	value   *[]int32
	changed bool
}

func newInt32SliceValue(val []int32, p *[]int32) *int32SliceValue {
	isv := new(int32SliceValue)
	isv.value = p
	*isv.value = val
	return isv
}

func (s *int32SliceValue) Set(val string) error {
	ss := strings.Split(val, ",")
	out := make([]int32, len(ss))
	for i, d := range ss {
		var err error
		var temp64 int64
		temp64, err = strconv.ParseInt(d, 0, 32)
		if err != nil {
			return err
		}
		out[i] = int32(temp64)

	}
	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}
	s.changed = true
	return nil
}

func (s *int32SliceValue) Type() string {
	return "int32Slice"
}

func (s *int32SliceValue) String() string {
	out := make([]string, len(*s.value))
	for i, d := range *s.value {
		out[i] = fmt.Sprintf("%d", d)
	}
	return "[" + strings.Join(out, ",") + "]"
}

func int32SliceConv(val string) (interface{}, error) {
	val = strings.Trim(val, "[]")
	// Empty string would cause a slice with one (empty) entry
	if len(val) == 0 {
		return []int32{}, nil
	}
	ss := strings.Split(val, ",")
	out := make([]int32, len(ss))
	for i, d := range ss {
		var err error
		var temp64 int64
		temp64, err = strconv.ParseInt(d, 0, 32)
		if err != nil {
			return nil, err
		}
		out[i] = int32(temp64)

	}
	return out, nil
}

// GetInt32Slice return the []int32 value of a flag with the given name
func (f *FlagSet) GetInt32Slice(name string) ([]int32, error) {
	val, err := f.getFlagType(name, "int32Slice", int32SliceConv)
	if err != nil {
		return []int32{}, err
	}
	return val.([]int32), nil
}

// Int32SliceVar defines a int32Slice flag with specified name, default value, and usage string.
// The argument p points to a []int32 variable in which to store the value of the flag.
func (f *FlagSet) Int32SliceVar(p *[]int32, name string, value []int32, usage string) {
	f.VarP(newInt32SliceValue(value, p), name, "", usage)
}

// Int32SliceVarP is like Int32SliceVar, but accepts a shorthand letter that can be used after a single dash.
func (f *FlagSet) Int32SliceVarP(p *[]int32, name, shorthand string, value []int32, usage string) {
	f.VarP(newInt32SliceValue(value, p), name, shorthand, usage)
}

// Int32SliceVar defines a int32[] flag with specified name, default value, and usage string.
// The argument p points to a int32[] variable in which to store the value of the flag.
func Int32SliceVar(p *[]int32, name string, value []int32, usage string) {
	CommandLine.VarP(newInt32SliceValue(value, p), name, "", usage)
}

// Int32SliceVarP is like Int32SliceVar, but accepts a shorthand letter that can be used after a single dash.
func Int32SliceVarP(p *[]int32, name, shorthand string, value []int32, usage string) {
	CommandLine.VarP(newInt32SliceValue(value, p), name, shorthand, usage)
}

// Int32Slice defines a []int32 flag with specified name, default value, and usage string.
// The return value is the address of a []int32 variable that stores the value of the flag.
func (f *FlagSet) Int32Slice(name string, value []int32, usage string) *[]int32 {
	p := []int32{}
	f.Int32SliceVarP(&p, name, "", value, usage)
	return &p
}

// Int32SliceP is like Int32Slice, but accepts a shorthand letter that can be used after a single dash.
func (f *FlagSet) Int32SliceP(name, shorthand string, value []int32, usage string) *[]int32 {
	p := []int32{}
	f.Int32SliceVarP(&p, name, shorthand, value, usage)
	return &p
}

// Int32Slice defines a []int32 flag with specified name, default value, and usage string.
// The return value is the address of a []int32 variable that stores the value of the flag.
func Int32Slice(name string, value []int32, usage string) *[]int32 {
	return CommandLine.Int32SliceP(name, "", value, usage)
}

// Int32SliceP is like Int32Slice, but accepts a shorthand letter that can be used after a single dash.
func Int32SliceP(name, shorthand string, value []int32, usage string) *[]int32 {
	return CommandLine.Int32SliceP(name, shorthand, value, usage)
}
