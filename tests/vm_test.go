// Copyright 2014 The go-etherfact Authors
// This file is part of the go-etherfact library.
//
// The go-etherfact library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherfact library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherfact library. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"testing"

	"github.com/EtherFact-Project/go-etherfact/core/vm"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vmt := new(testMatcher)
	vmt.fails("^vmSystemOperationsTest.json/createNameRegistrator$", "fails without parallel execution")

	vmt.skipLoad(`^vmInputLimits(Light)?.json`) // log format broken

	vmt.skipShortMode("^vmPerformanceTest.json")
	vmt.skipShortMode("^vmInputLimits(Light)?.json")

	vmt.walk(t, vmTestDir, func(t *testing.T, name string, test *VMTest) {
		withTrace(t, test.json.Exec.GasLimit, func(vmconfig vm.Config) error {
			return vmt.checkFailure(t, name, test.Run(vmconfig))
		})
	})
}
