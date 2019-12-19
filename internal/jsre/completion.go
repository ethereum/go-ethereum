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

package jsre

import (
	"sort"
	"strings"

	"github.com/dop251/goja"
)

// CompleteKeywords returns potential continuations for the given line. Since line is
// evaluated, callers need to make sure that evaluating line does not have side effects.
func (jsre *JSRE) CompleteKeywords(line string) []string {
	var results []string
	jsre.Do(func(vm *goja.Runtime) {
		results = getCompletions(vm, line)
	})
	return results
}

func getCompletions(vm *goja.Runtime, line string) (results []string) {
	parts := strings.Split(line, ".")
	objRef := "this"
	prefix := line

	if len(parts) == 0 {
		return nil
	}

	// Figure out which is the right-most fully named object
	// in the line. e.g. if line = "x.y.z" and "x.y" is an
	// object, and that its keys are "zebu" and "zebra", then
	// objRef will be set to "y" and obj will reference "x.y".
	v := vm.Get(parts[0])
	var obj *goja.Object = v.ToObject(vm)
	switch {
	case obj != nil && len(parts) > 1: // "x.y.z" case
		objRef = strings.Join(parts[0:len(parts)-1], ".")
		prefix = parts[len(parts)-1]
		for _, part := range parts[1 : len(parts)-1] {
			v = obj.Get(part)
			if v == nil {
				return nil
			}
			obj = v.ToObject(vm)
		}
	case obj != nil:
		// In this case, there is no "." chain, so the
		// the right-most object is assumed to be `this`.
		obj = vm.GlobalObject()
	default: // No object was found
		return nil
	}

	// Go over the keys of the right-most object (which could
	// be `this`) and retain those keys that are prefixed by
	// `prefix`. e.g. if line = "x.y.z", that "x.y" exists
	// and has keys "zebu", "zebra" and "platypus", then only
	// "zebu" and "zebra" will be added to `results`.
	iterOwnAndConstructorKeys(vm, obj, func(k string) {
		if strings.HasPrefix(k, prefix) {
			if objRef == "this" {
				results = append(results, line)
			} else {
				results = append(results, strings.Join(parts[:len(parts)-1], ".")+"."+k)
			}
		}
	})

	// Append opening parenthesis (for functions) or dot (for objects)
	// if the line itself is the only completion.
	if len(results) == 1 && results[0] == line {
		obj := obj.Get(parts[len(parts)-1])
		if obj != nil {
			if _, isfunc := goja.AssertFunction(obj); isfunc {
				results[0] += "("
			} else {
				results[0] += "."
			}
		}
	}

	sort.Strings(results)
	return results
}
