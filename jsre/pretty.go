// Copyright 2015 The go-ethereum Authors
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

package jsre

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/robertkrimen/otto"
)

const (
	maxPrettyPrintLevel = 3
	indentString        = "  "
)

var (
	functionColor = color.New(color.FgMagenta)
	specialColor  = color.New(color.Bold)
	numberColor   = color.New(color.FgRed)
	stringColor   = color.New(color.FgGreen)
)

// these fields are hidden when printing objects.
var boringKeys = map[string]bool{
	"valueOf":              true,
	"toString":             true,
	"toLocaleString":       true,
	"hasOwnProperty":       true,
	"isPrototypeOf":        true,
	"propertyIsEnumerable": true,
	"constructor":          true,
}

// prettyPrint writes value to standard output.
func prettyPrint(vm *otto.Otto, value otto.Value) {
	ppctx{vm}.printValue(value, 0, false)
}

func prettyPrintJS(call otto.FunctionCall) otto.Value {
	for _, v := range call.ArgumentList {
		prettyPrint(call.Otto, v)
		fmt.Println()
	}
	return otto.UndefinedValue()
}

type ppctx struct{ vm *otto.Otto }

func (ctx ppctx) indent(level int) string {
	return strings.Repeat(indentString, level)
}

func (ctx ppctx) printValue(v otto.Value, level int, inArray bool) {
	switch {
	case v.IsObject():
		ctx.printObject(v.Object(), level, inArray)
	case v.IsNull():
		specialColor.Print("null")
	case v.IsUndefined():
		specialColor.Print("undefined")
	case v.IsString():
		s, _ := v.ToString()
		stringColor.Printf("%q", s)
	case v.IsBoolean():
		b, _ := v.ToBoolean()
		specialColor.Printf("%t", b)
	case v.IsNaN():
		numberColor.Printf("NaN")
	case v.IsNumber():
		s, _ := v.ToString()
		numberColor.Printf("%s", s)
	default:
		fmt.Printf("<unprintable>")
	}
}

func (ctx ppctx) printObject(obj *otto.Object, level int, inArray bool) {
	switch obj.Class() {
	case "Array":
		lv, _ := obj.Get("length")
		len, _ := lv.ToInteger()
		if len == 0 {
			fmt.Printf("[]")
			return
		}
		if level > maxPrettyPrintLevel {
			fmt.Print("[...]")
			return
		}
		fmt.Print("[")
		for i := int64(0); i < len; i++ {
			el, err := obj.Get(strconv.FormatInt(i, 10))
			if err == nil {
				ctx.printValue(el, level+1, true)
			}
			if i < len-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Print("]")

	case "Object":
		// Print values from bignumber.js as regular numbers.
		if ctx.isBigNumber(obj) {
			numberColor.Print(toString(obj))
			return
		}
		// Otherwise, print all fields indented, but stop if we're too deep.
		keys := ctx.fields(obj)
		if len(keys) == 0 {
			fmt.Print("{}")
			return
		}
		if level > maxPrettyPrintLevel {
			fmt.Print("{...}")
			return
		}
		fmt.Println("{")
		for i, k := range keys {
			v, _ := obj.Get(k)
			fmt.Printf("%s%s: ", ctx.indent(level+1), k)
			ctx.printValue(v, level+1, false)
			if i < len(keys)-1 {
				fmt.Printf(",")
			}
			fmt.Println()
		}
		if inArray {
			level--
		}
		fmt.Printf("%s}", ctx.indent(level))

	case "Function":
		// Use toString() to display the argument list if possible.
		if robj, err := obj.Call("toString"); err != nil {
			functionColor.Print("function()")
		} else {
			desc := strings.Trim(strings.Split(robj.String(), "{")[0], " \t\n")
			desc = strings.Replace(desc, " (", "(", 1)
			functionColor.Print(desc)
		}

	case "RegExp":
		stringColor.Print(toString(obj))

	default:
		if v, _ := obj.Get("toString"); v.IsFunction() && level <= maxPrettyPrintLevel {
			s, _ := obj.Call("toString")
			fmt.Printf("<%s %s>", obj.Class(), s.String())
		} else {
			fmt.Printf("<%s>", obj.Class())
		}
	}
}

func (ctx ppctx) fields(obj *otto.Object) []string {
	var (
		vals, methods []string
		seen          = make(map[string]bool)
	)
	add := func(k string) {
		if seen[k] || boringKeys[k] {
			return
		}
		seen[k] = true
		if v, _ := obj.Get(k); v.IsFunction() {
			methods = append(methods, k)
		} else {
			vals = append(vals, k)
		}
	}
	iterOwnAndConstructorKeys(ctx.vm, obj, add)
	sort.Strings(vals)
	sort.Strings(methods)
	return append(vals, methods...)
}

func iterOwnAndConstructorKeys(vm *otto.Otto, obj *otto.Object, f func(string)) {
	seen := make(map[string]bool)
	iterOwnKeys(vm, obj, func(prop string) {
		seen[prop] = true
		f(prop)
	})
	if cp := constructorPrototype(obj); cp != nil {
		iterOwnKeys(vm, cp, func(prop string) {
			if !seen[prop] {
				f(prop)
			}
		})
	}
}

func iterOwnKeys(vm *otto.Otto, obj *otto.Object, f func(string)) {
	Object, _ := vm.Object("Object")
	rv, _ := Object.Call("getOwnPropertyNames", obj.Value())
	gv, _ := rv.Export()
	switch gv := gv.(type) {
	case []interface{}:
		for _, v := range gv {
			f(v.(string))
		}
	case []string:
		for _, v := range gv {
			f(v)
		}
	default:
		panic(fmt.Errorf("Object.getOwnPropertyNames returned unexpected type %T", gv))
	}
}

func (ctx ppctx) isBigNumber(v *otto.Object) bool {
	BigNumber, err := ctx.vm.Run("BigNumber.prototype")
	if err != nil {
		panic(err)
	}
	cp := constructorPrototype(v)
	return cp != nil && cp.Value() == BigNumber
}

func toString(obj *otto.Object) string {
	s, _ := obj.Call("toString")
	return s.String()
}

func constructorPrototype(obj *otto.Object) *otto.Object {
	if v, _ := obj.Get("constructor"); v.Object() != nil {
		if v, _ = v.Object().Get("prototype"); v.Object() != nil {
			return v.Object()
		}
	}
	return nil
}
