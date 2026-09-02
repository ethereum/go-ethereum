// Copyright 2026 The go-ethereum Authors
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

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/google/pprof/profile"
)

// This file writes the PGO profile that gets the opcode handlers inlined into
// the generated dispatch.
//
// The dispatch calls its handlers rather than splicing their bodies, and their
// bodies cost far more than the compiler's inline budget of 80, so without a
// profile they stay real calls. Capturing one by running blocks works but has
// two problems. It only covers opcodes the sampled blocks happened to execute,
// so the rest keep their calls. And PGO identifies a call site by its line
// offset from the start of the enclosing function, so regenerating the dispatch
// moves all of them at once and the profile silently stops matching, which
// looks like a performance regression and is not one.
//
// Nothing in a profile is needed here except the caller, the callee, and where
// the call is, all of which this generator knows because it wrote the file. So
// it writes the profile too, and the two cannot drift apart.
//
// This is not a measurement. It is an inlining directive in a profile's format,
// and it says so rather than pretending otherwise. A captured profile can still
// be merged in if the wider program is worth optimizing, but it has to carry the
// same sample types or the merge silently drops one side.

// pgoWeight is the weight given to every call site. The absolute value does not
// matter, only that these edges sit above the profile's hot cutoff.
const pgoWeight = 1 << 20

// buildProfile returns a profile marking every handler call in the generated
// dispatch as hot. src is the formatted output, which is parsed rather than
// tracked while emitting because gofmt can move lines and the offsets have to
// match the file that ships.
func (g *generator) buildProfile(src []byte) *profile.Profile {
	startLine, sites := g.callSites(src)
	if len(sites) == 0 {
		abortf("no handler calls found in the generated dispatch, so the profile would inline nothing")
	}

	p := &profile.Profile{
		// Both sample types, matching what runtime/pprof emits. A merge of two
		// profiles whose sample types differ drops one of them without saying so.
		SampleType: []*profile.ValueType{
			{Type: "samples", Unit: "count"},
			{Type: "cpu", Unit: "nanoseconds"},
		},
		PeriodType: &profile.ValueType{Type: "cpu", Unit: "nanoseconds"},
		Period:     1,
	}
	// StartLine is what the call site offset is measured from. The compiler
	// rejects a profile that omits it.
	caller := &profile.Function{
		ID:         1,
		Name:       vmPkgPath + "." + dispatchName,
		SystemName: vmPkgPath + "." + dispatchName,
		Filename:   generatedFile,
		StartLine:  int64(startLine),
	}
	p.Function = append(p.Function, caller)

	id := uint64(2)
	loc := func(fn *profile.Function, line int) *profile.Location {
		l := &profile.Location{ID: id, Line: []profile.Line{{Function: fn, Line: int64(line)}}}
		id++
		p.Location = append(p.Location, l)
		return l
	}
	var first *profile.Location
	for _, s := range sites {
		name := vmPkgPath + "." + s.callee
		callee := &profile.Function{ID: id, Name: name, SystemName: name, Filename: handlerFile, StartLine: 1}
		id++
		p.Function = append(p.Function, callee)

		// Leaf frame first. Only the callee's name matters there, and the caller
		// frame carries the line the call is on.
		leaf := loc(callee, 1)
		at := loc(caller, s.line)
		if first == nil {
			first = leaf
		}
		p.Sample = append(p.Sample, &profile.Sample{
			Location: []*profile.Location{leaf, at},
			Value:    []int64{pgoWeight, pgoWeight},
		})
	}

	// The hot cutoff is a cumulative threshold: edges are ranked by weight and
	// accumulated until 99% of the total is covered, and the weight where that
	// lands becomes the bar. Equal weights put the bar on that weight and the
	// comparison is strict, so nothing clears it and the profile does nothing.
	// A tail of cheap edges holds the distribution open. They point at call site
	// offsets the dispatch does not have, so they match nothing.
	tail := len(sites)*11/10 + 10
	tailWeight := max(int64(1), pgoWeight*int64(len(sites))*2/100/int64(tail))
	for i := range tail {
		at := loc(caller, startLine+1e6+i)
		p.Sample = append(p.Sample, &profile.Sample{
			Location: []*profile.Location{first, at},
			Value:    []int64{tailWeight, tailWeight},
		})
	}
	return p
}

// handlerSite is one handler call in the dispatch.
type handlerSite struct {
	callee string
	line   int
}

// callSites parses the formatted dispatch and returns the line the dispatch
// function starts on plus every handler it calls by name. Handlers are called
// as bare identifiers, so anything selected off a value is a method or a table
// pointer and is not something the profile can name.
func (g *generator) callSites(src []byte) (startLine int, sites []handlerSite) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, generatedFile, src, 0)
	if err != nil {
		abortf("parsing the generated dispatch to find its call sites: %v", err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name.Name != dispatchFunc {
			continue
		}
		startLine = fset.Position(fn.Pos()).Line
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || !strings.HasPrefix(id.Name, "op") {
				return true
			}
			sites = append(sites, handlerSite{callee: id.Name, line: fset.Position(call.Lparen).Line})
			return true
		})
	}
	return startLine, sites
}

// profileFor builds the profile for a generated dispatch, turning a tripped
// guard into an error the way generate does.
func profileFor(src []byte) (p *profile.Profile, err error) {
	defer func() {
		switch r := recover().(type) {
		case nil:
		case genError:
			err = r
		default:
			panic(r)
		}
	}()
	g := &generator{}
	return g.buildProfile(src), nil
}
