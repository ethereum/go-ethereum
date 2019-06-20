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
	var obj *goja.Object
	if len(parts) > 1 {
		objRef = strings.Join(parts[0:len(parts)-1], ".")
		prefix = parts[len(parts)-1]
		obj, _ = vm.Get(objRef).(*goja.Object)
	} else {
		obj = vm.GlobalObject()
	}

	if obj == nil {
		return nil
	}
	iterOwnAndConstructorKeys(vm, obj, func(k string) {
		if strings.HasPrefix(k, prefix) {
			if objRef == "this" {
				results = append(results, k)
			} else {
				results = append(results, strings.Join(parts[:len(parts)-1], ".")+"."+k)
			}
		}
	})

	// Append opening parenthesis (for functions) or dot (for objects)
	// if the line itself is the only completion.
	if len(results) == 1 && results[0] == line {
		/* XXX Get will return `nil` et j'avais suppose que ca lancerait
		une exception donc je dois tout revoir */
		obj := vm.Get(line)
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
