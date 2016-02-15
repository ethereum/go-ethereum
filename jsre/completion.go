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
	"sort"
	"strings"

	"github.com/robertkrimen/otto"
)

// CompleteKeywords returns potential continuations for the given line. Since line is
// evaluated, callers need to make sure that evaluating line does not have side effects.
func (jsre *JSRE) CompleteKeywords(line string) []string {
	var results []string
	jsre.do(func(vm *otto.Otto) { results = getCompletions(vm, line) })
	return results
}

func getCompletions(vm *otto.Otto, line string) (results []string) {
	parts := strings.Split(line, ".")
	objRef := "this"
	prefix := line
	if len(parts) > 1 {
		objRef = strings.Join(parts[0:len(parts)-1], ".")
		prefix = parts[len(parts)-1]
	}

	obj, _ := vm.Object(objRef)
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
	// e.g. web3<tab><tab> append dot since its an object
	if obj, _ = vm.Object(line); obj != nil {
		results = append(results, line+".")
	}

	sort.Strings(results)
	return results
}
