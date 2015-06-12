/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */

package ethash

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/logger/glog"
)

func TestGPUMiner(t *testing.T) {
	glog.SetV(6)
	glog.SetToStderr(true)

	fmt.Println("initialising OpenCL miner...")

	gpu, err := NewEthashOpenCL(30002, 4)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("OpenCL miner initialised")

	hash := rndHash()
	fmt.Printf("Searching for solution for %x\n", hash)
	err = gpu.DebugSearch(hash, math.MaxUint64/(1024*512), func(res []uint64) bool {
		fmt.Printf("found: %x\n", res)
		return true
	})
	if err != nil {
		fmt.Println("search err:", err)
		os.Exit(1)
	}
	gpu.ctx.Release()
	// TODO: release dag buffs
	gpu.headerBuff.Release()
	gpu.searchKernel.Release()
	gpu.queue.Release()
	for _, searchBuffer := range gpu.searchBuffers {
		searchBuffer.Release()
	}
}
