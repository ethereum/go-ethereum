// Copyright 2024 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import "regexp"

// testMetadata provides more granular access to the test information encoded
// within its filename by the execution spec test (EEST).
type testMetadata struct {
	fork       string
	module     string // which python module generated the test, e.g. eip7702
	file       string // exact file the test came from, e.g. test_gas.py
	function   string // func that created the test, e.g. test_valid_mcopy_operations
	parameters string // the name of the parameters which were used to fill the test, e.g. zero_inputs
}

// parseTestMetadata reads a test name and parses out more specific information
// about the test.
func parseTestMetadata(s string) *testMetadata {
	var (
		pattern = `tests\/([^\/]+)\/([^\/]+)\/([^:]+)::([^[]+)\[fork_([^-\]]+)-[^-]+-(.+)\]`
		re      = regexp.MustCompile(pattern)
	)
	match := re.FindStringSubmatch(s)
	if len(match) == 0 {
		return nil
	}
	return &testMetadata{
		fork:       match[5],
		module:     match[2],
		file:       match[3],
		function:   match[4],
		parameters: match[6],
	}
}
