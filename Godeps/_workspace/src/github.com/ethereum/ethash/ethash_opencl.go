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

/*

  TODO: This code needs a lot of cleanup & refactoring! /Gustav

  In summary, this code have two main parts:

  1. The initCL(...)  function configures one or more OpenCL device
     (for now only GPU) and loads the Ethash DAG into device memory

  2. The Search(...) function loads a Ethash nonce into device(s) memory and
     executes the Ethash OpenCL kernel.

  Throughout the code, we refer to "host memory" and "device memory".
  For most systems (e.g. regular PC GPU miner) the host memory is RAM and
  device memory is the GPU global memory (e.g. GDDR5).

  References are refered to by [1], [3] etc.

  1. https://github.com/ethereum/wiki/wiki/Ethash
  2. https://github.com/ethereum/cpp-ethereum/blob/develop/libethash-cl/ethash_cl_miner.cpp
  3. https://www.khronos.org/registry/cl/sdk/1.2/docs/man/xhtml/
  4. http://amd-dev.wpengine.netdna-cdn.com/wordpress/media/2013/12/AMD_OpenCL_Programming_User_Guide.pdf

*/

package ethash

//#cgo LDFLAGS: -w
//#include <stdint.h>
//#include <string.h>
//#include "src/libethash/internal.h"
import "C"

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/Gustav-Simonsson/go-opencl/cl"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/pow"
)

type OpenCLDevice struct {
	deviceId int
	device   *cl.Device
	openCL11 bool // sometimes we need to check OpenCL version
	openCL12 bool

	dagChunks []*cl.MemObject // DAG in device mem

	headerBuff    *cl.MemObject // Hash of block-to-mine in device mem
	searchBuffers []*cl.MemObject

	searchKernel *cl.Kernel
	hashKernel   *cl.Kernel

	queue         *cl.CommandQueue
	ctx           *cl.Context
	workGroupSize int

	nonceRand *mrand.Rand
	result    common.Hash
}

type OpenCLMiner struct {
	mu sync.Mutex

	ethash *Ethash // Ethash full DAG & cache in host mem

	deviceIds []int
	devices   []*OpenCLDevice

	dagSize      uint64
	dagChunksNum uint64

	hashRate int32 // Go atomics & uint64 have some issues, int32 is supported on all platforms
}

type pendingSearch struct {
	bufIndex   uint32
	startNonce uint64
}

const (
	SIZEOF_UINT32 = 4

	// See [1]
	ethashMixBytesLen = 128
	ethashAccesses    = 64

	// See [4]
	workGroupSize    = 32 // must be multiple of 8
	maxSearchResults = 63
	searchBuffSize   = 2
	globalWorkSize   = 1024 * 256

	//gpuMemMargin = 1024 * 1024 * 512

	// TODO: config flags for these
	//checkGpuMemMargin = true
)

func NewCL(dagChunksNum uint64, deviceIds []int) *OpenCLMiner {
	ids := make([]int, len(deviceIds))
	copy(ids, deviceIds)
	return &OpenCLMiner{
		ethash:       New(),
		dagChunksNum: dagChunksNum,
		dagSize:      0,
		deviceIds:    ids,
	} // dagSize is used to see if we need to update DAG.
}

/* See [2]. We basically do the same here, but the Go OpenCL bindings
   are at a slightly higher abtraction level.
*/
// TODO: proper solution for automatic DAG switch at epoch change
func PrintDevices() {
	fmt.Println("=============================================")
	fmt.Println("============ OpenCL Device Info =============")
	fmt.Println("=============================================")

	var found []*cl.Device

	platforms, err := cl.GetPlatforms()
	if err != nil {
		fmt.Println("Plaform error (check your OpenCL installation): %v", err)
		return
	}

	for i, p := range platforms {
		fmt.Println("Platform id             ", i)
		fmt.Println("Platform Name           ", p.Name())
		fmt.Println("Platform Vendor         ", p.Vendor())
		fmt.Println("Platform Version        ", p.Version())
		fmt.Println("Platform Extensions     ", p.Extensions())
		fmt.Println("Platform Profile        ", p.Profile())
		fmt.Println("")

		devices, err := cl.GetDevices(p, cl.DeviceTypeGPU)
		if err != nil {
			fmt.Println("Device error (check your GPU drivers) :", err)
			return
		}

		for _, d := range devices {
			fmt.Println("Device OpenCL id        ", i)
			fmt.Println("Device id for mining    ", len(found))
			fmt.Println("Device Name             ", d.Name())
			fmt.Println("Vendor                  ", d.Vendor())
			fmt.Println("Version                 ", d.Version())
			fmt.Println("Driver version          ", d.DriverVersion())
			fmt.Println("Address bits            ", d.AddressBits())
			fmt.Println("Max clock freq          ", d.MaxClockFrequency())
			fmt.Println("Global mem size         ", d.GlobalMemSize())
			fmt.Println("Max constant buffer size", d.MaxConstantBufferSize())
			fmt.Println("Max mem alloc size      ", d.MaxMemAllocSize())
			fmt.Println("Max compute units       ", d.MaxComputeUnits())
			fmt.Println("Max work group size     ", d.MaxWorkGroupSize())
			fmt.Println("Max work item sizes     ", d.MaxWorkItemSizes())
			fmt.Println("=============================================")

			found = append(found, d)
		}
	}
	if len(found) == 0 {
		fmt.Println("Found no GPU(s). Check that your OS can see the GPU(s)")
	} else {
		var idsFormat string
		for i := 0; i < len(found); i++ {
			idsFormat += strconv.Itoa(i)
			if i != len(found)-1 {
				idsFormat += ","
			}
		}
		fmt.Printf("Found %v devices. Benchmark first GPU:       geth gpubench 0\n", len(found))
		fmt.Printf("Mine using all GPUs:                        geth --minegpu %v\n", idsFormat)
		fmt.Println("You may also need to enable chunking:            --gpuchunks")
	}
}

func InitCL(blockNum uint64, c *OpenCLMiner) error {
	if !(c.dagChunksNum == 1 || c.dagChunksNum == 4) {
		return fmt.Errorf("DAG chunks num must be 1 or 4")
	}

	platforms, err := cl.GetPlatforms()
	if err != nil {
		return fmt.Errorf("Plaform error: %v\nCheck your OpenCL installation and then run geth gpuinfo", err)
	}

	var devices []*cl.Device
	for _, p := range platforms {
		ds, err := cl.GetDevices(p, cl.DeviceTypeGPU)
		if err != nil {
			return fmt.Errorf("Devices error: %v\nCheck your GPU drivers and then run geth gpuinfo", err)
		}
		for _, d := range ds {
			devices = append(devices, d)
		}
	}

	pow := New()
	// generates DAG if we don't have it
	_ = pow.getDAG(blockNum)
	// and cache. TODO: unfuck
	pow.Light.getCache(blockNum)

	c.ethash = pow
	dagSize := pow.getDAGSize(blockNum)
	c.dagSize = dagSize

	for _, id := range c.deviceIds {
		if id > len(devices)-1 {
			return fmt.Errorf("Device id not found. See available device ids with: geth gpuinfo")
		} else {
			err := initCLDevice(id, devices[id], c)
			if err != nil {
				return err
			}
		}
	}
	if len(c.devices) == 0 {
		return fmt.Errorf("No GPU devices found")
	}
	return nil
}

func initCLDevice(deviceId int, device *cl.Device, c *OpenCLMiner) error {
	devMaxAlloc := uint64(device.MaxMemAllocSize())
	devGlobalMem := uint64(device.GlobalMemSize())

	// TODO: more fine grained version logic
	if device.Version() == "OpenCL 1.0" {
		fmt.Println("Device OpenCL version not supported: ", device.Version())
		return fmt.Errorf("opencl version not supported")
	}

	var cl11, cl12 bool
	if device.Version() == "OpenCL 1.1" {
		cl11 = true
	}
	if device.Version() == "OpenCL 1.2" {
		cl12 = true
	}

	if c.dagSize > devGlobalMem {
		fmt.Printf("WARNING: device memory may be insufficient: %v. DAG size: %v. You may have to run with --gpuchunks\n", devGlobalMem, c.dagSize)
		// TODO: we continue since it seems sometimes clGetDeviceInfo reports wrong numbers
		//return fmt.Errorf("Insufficient device memory")
	}

	if c.dagSize > devMaxAlloc {
		fmt.Printf("WARNING: DAG size (%v) larger than device max memory allocation size (%v).\n", c.dagSize, devMaxAlloc)
		fmt.Printf("You may have to run with --gpuchunks\n")
		//return fmt.Errorf("Insufficient device memory")
	}

	fmt.Printf("Initialising device %v: %v\n", deviceId, device.Name())

	context, err := cl.CreateContext([]*cl.Device{device})
	if err != nil {
		return fmt.Errorf("failed creating context:", err)
	}

	// TODO: test running with CL_QUEUE_PROFILING_ENABLE for profiling?
	queue, err := context.CreateCommandQueue(device, 0)
	if err != nil {
		return fmt.Errorf("command queue err:", err)
	}

	// See [4] section 3.2 and [3] "clBuildProgram".
	// The OpenCL kernel code is compiled at run-time.
	kvs := make(map[string]string, 4)
	kvs["GROUP_SIZE"] = strconv.FormatUint(workGroupSize, 10)
	kvs["DAG_SIZE"] = strconv.FormatUint(c.dagSize/ethashMixBytesLen, 10)
	kvs["ACCESSES"] = strconv.FormatUint(ethashAccesses, 10)
	kvs["MAX_OUTPUTS"] = strconv.FormatUint(maxSearchResults, 10)
	kernelCode := replaceWords(kernel, kvs)

	program, err := context.CreateProgramWithSource([]string{kernelCode})
	if err != nil {
		return fmt.Errorf("program err:", err)
	}

	/* if using AMD OpenCL impl, you can set this to debug on x86 CPU device.
	   see AMD OpenCL programming guide section 4.2

	   export in shell before running:
	   export AMD_OCL_BUILD_OPTIONS_APPEND="-g -O0"
	   export CPU_MAX_COMPUTE_UNITS=1

	buildOpts := "-g -cl-opt-disable"

	*/
	buildOpts := ""
	err = program.BuildProgram([]*cl.Device{device}, buildOpts)
	if err != nil {
		return fmt.Errorf("program build err:", err)
	}

	var searchKernelName, hashKernelName string
	if c.dagChunksNum == 4 {
		searchKernelName = "ethash_search_chunks"
		hashKernelName = "ethash_hash_chunks"
	} else {
		searchKernelName = "ethash_search"
		hashKernelName = "ethash_hash"
	}

	searchKernel, err := program.CreateKernel(searchKernelName)
	hashKernel, err := program.CreateKernel(hashKernelName)
	if err != nil {
		return fmt.Errorf("kernel err:", err)
	}

	// TODO: in case chunk allocation is not default when this DAG size appears, patch
	// the Go bindings (context.go) to work with uint64 as size_t
	if c.dagSize > math.MaxInt32 {
		fmt.Println("DAG too large for non-chunk allocation. Try running with --opencl-mem-chunking")
		return fmt.Errorf("DAG too large for non-chunk alloc")
	}

	chunkSize := func(i uint64) uint64 {
		if c.dagChunksNum == 4 {
			if i == 3 {
				return c.dagSize - 3*((c.dagSize>>9)<<7)
			} else {
				return (c.dagSize >> 9) << 7
			}
		} else {
			return c.dagSize
		}
	}

	// allocate device mem
	dagChunks := make([]*cl.MemObject, c.dagChunksNum)
	for i := uint64(0); i < c.dagChunksNum; i++ {
		// TODO: patch up Go bindings to work with size_t, chunkSize will overflow if > maxint32
		// TODO: fuck. shit's gonna overflow soon
		dagChunk, err := context.CreateEmptyBuffer(cl.MemReadOnly, int(chunkSize(i)))
		if err != nil {
			return fmt.Errorf("allocating dag chunks failed: ", err)
		}
		dagChunks[i] = dagChunk
	}

	// write DAG to device mem
	var offset uint64
	for i := uint64(0); i < c.dagChunksNum; i++ {
		offset = chunkSize(0) * i
		size := chunkSize(i)
		dagPtr := uintptr(unsafe.Pointer(c.ethash.Full.current.ptr.data))
		offsetPtr := unsafe.Pointer(dagPtr + uintptr(offset))
		fmt.Println("OpenCL EnqueueWriteBuffer (DAG): host mem offset, chunkSize, dagSize: ", offset, size, c.dagSize)
		// offset into device buffer is always 0, offset into DAG depends on chunk
		if c.dagChunksNum == 1 {
			_, err = queue.EnqueueWriteBuffer(dagChunks[i], true, 0, int(size), offsetPtr, nil)
			if err != nil {
				return fmt.Errorf("writing to dag chunk failed: ", err)
			}
		} else {
			// TODO: replace with EnqueueWriteBuffer
			mmo, _, err := queue.EnqueueMapBuffer(dagChunks[i], true, cl.MapFlagWrite, 0, int(size), nil)
			if err != nil {
				fmt.Println("Error in Search clEnqueueMapBuffer: ", err)
				return fmt.Errorf("mapping buffer for DAG chunk write failed: ", err)
			}
			C.memcpy(mmo.Ptr(), offsetPtr, C.size_t(size))
			_, err = queue.EnqueueUnmapMemObject(dagChunks[i], mmo, nil)
			if err != nil {
				fmt.Println("Error in Search clEnqueueUnmapMemObject: ", err)
				return fmt.Errorf("unmapping buffer after DAG chunk write failed: ", err)
			}
		}
	}

	searchBuffers := make([]*cl.MemObject, searchBuffSize)
	for i := 0; i < searchBuffSize; i++ {
		searchBuff, err := context.CreateEmptyBuffer(cl.MemWriteOnly, (1+maxSearchResults)*SIZEOF_UINT32)
		if err != nil {
			return fmt.Errorf("search buffer err:", err)
		}
		searchBuffers[i] = searchBuff
	}

	// hash of block-to-mine in device mem
	headerBuff, err := context.CreateEmptyBuffer(cl.MemReadOnly, 32)
	if err != nil {
		return fmt.Errorf("header buffer err:", err)
	}

	// Unique, random nonces are crucial for mining efficieny.
	// While we do not need cryptographically secure PRNG for nonces,
	// we want to have uniform distribution and minimal repetition of nonces.
	// We could guarantee strict uniqueness of nonces by generating unique ranges,
	// but a int64 seed from crypto/rand should be good enough.
	// we then use math/rand for speed and to avoid draining OS entropy pool
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return err
	}
	nonceRand := mrand.New(mrand.NewSource(seed.Int64()))

	deviceStruct := &OpenCLDevice{
		deviceId: deviceId,
		device:   device,
		openCL11: cl11,
		openCL12: cl12,

		dagChunks:     dagChunks,
		headerBuff:    headerBuff,
		searchBuffers: searchBuffers,

		searchKernel: searchKernel,
		hashKernel:   hashKernel,

		queue: queue,
		ctx:   context,

		workGroupSize: workGroupSize,

		nonceRand: nonceRand,
	}
	c.devices = append(c.devices, deviceStruct)

	return nil
}

func (c *OpenCLMiner) Search(block pow.Block, stop <-chan struct{}, index int) (uint64, []byte) {
	c.mu.Lock()
	newDagSize := c.ethash.getDAGSize(block.NumberU64())
	if newDagSize > c.dagSize {
		// TODO: clean up buffers from previous DAG?
		err := InitCL(block.NumberU64(), c)
		if err != nil {
			fmt.Println("OpenCL init error: ", err)
			return 0, []byte{0}
		}
	}
	defer c.mu.Unlock()

	// Avoid unneeded OpenCL initialisation if we received stop while running InitCL
	select {
	case <-stop:
		return 0, []byte{0}
	default:
	}

	headerHash := block.HashNoNonce()
	//headerHash := common.HexToHash("b832154e35c5480afda424509c49885fcf23d1467a375e24929de07226993c77")
	diff := block.Difficulty()
	target256 := new(big.Int).Div(MaxUint256, diff)
	target64 := new(big.Int).Rsh(target256, 192).Uint64()
	var zero uint32 = 0

	d := c.devices[index]

	_, err := d.queue.EnqueueWriteBuffer(d.headerBuff, false, 0, 32, unsafe.Pointer(&headerHash[0]), nil)
	if err != nil {
		fmt.Println("Error in Search clEnqueueWriterBuffer : ", err)
		return 0, []byte{0}
	}

	for i := 0; i < searchBuffSize; i++ {
		_, err := d.queue.EnqueueWriteBuffer(d.searchBuffers[i], false, 0, 4, unsafe.Pointer(&zero), nil)
		if err != nil {
			fmt.Println("Error in Search clEnqueueWriterBuffer : ", err)
			return 0, []byte{0}
		}
	}

	// wait for all search buffers to complete
	err = d.queue.Finish()
	if err != nil {
		fmt.Println("Error in Search clFinish : ", err)
		return 0, []byte{0}
	}

	err = d.searchKernel.SetArg(1, d.headerBuff)
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}

	argPos := 2
	for i := uint64(0); i < c.dagChunksNum; i++ {
		err = d.searchKernel.SetArg(argPos, d.dagChunks[i])
		if err != nil {
			fmt.Println("Error in Search clSetKernelArg : ", err)
			return 0, []byte{0}
		}
		argPos++
	}
	err = d.searchKernel.SetArg(argPos+1, target64)
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}
	err = d.searchKernel.SetArg(argPos+2, uint32(math.MaxUint32))
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}

	// we wait for this one before returning
	var preReturnEvent *cl.Event
	if d.openCL12 {
		preReturnEvent, err = d.ctx.CreateUserEvent()
		if err != nil {
			fmt.Println("Error in Search create CL user event : ", err)
			return 0, []byte{0}
		}
	}

	pending := make([]pendingSearch, 0, searchBuffSize)
	var p *pendingSearch
	searchBufIndex := uint32(0)
	var checkNonce uint64
	loops := int64(0)
	prevHashRate := int32(0)
	start := time.Now().UnixNano()
	// we grab a single random nonce and sets this as argument to the kernel search function
	// the device will then add each local threads gid to the nonce, creating a unique nonce
	// for each device computing unit executing in parallel
	initNonce := uint64(d.nonceRand.Int63())
	for nonce := initNonce; ; nonce += uint64(globalWorkSize) {
		select {
		case <-stop:

			/*
				if d.openCL12 {
					err = cl.WaitForEvents([]*cl.Event{preReturnEvent})
					if err != nil {
						fmt.Println("Error in Search WaitForEvents: ", err)
					}
				}
			*/

			atomic.AddInt32(&c.hashRate, -prevHashRate)
			return 0, []byte{0}
		default:
		}

		if (loops % (1 << 7)) == 0 {
			elapsed := time.Now().UnixNano() - start
			hashes := (float64(1e9) / float64(elapsed)) * float64(loops*1024*256)
			hashrateDiff := int32(hashes) - prevHashRate
			prevHashRate = int32(hashes)
			atomic.AddInt32(&c.hashRate, hashrateDiff)
		}
		loops++

		err = d.searchKernel.SetArg(0, d.searchBuffers[searchBufIndex])
		if err != nil {
			fmt.Println("Error in Search clSetKernelArg : ", err)
			return 0, []byte{0}
		}
		// TODO: only works with 1 or 4 currently
		var argPos int
		if c.dagChunksNum == 1 {
			argPos = 3
		} else if c.dagChunksNum == 4 {
			argPos = 6
		}
		err = d.searchKernel.SetArg(argPos, nonce)
		if err != nil {
			fmt.Println("Error in Search clSetKernelArg : ", err)
			return 0, []byte{0}
		}

		// executes kernel; either the "ethash_search" or "ethash_search_chunks" function
		_, err := d.queue.EnqueueNDRangeKernel(
			d.searchKernel,
			[]int{0},
			[]int{globalWorkSize},
			[]int{d.workGroupSize},
			nil)
		if err != nil {
			fmt.Println("Error in Search clEnqueueNDRangeKernel : ", err)
			return 0, []byte{0}
		}

		pending = append(pending, pendingSearch{bufIndex: searchBufIndex, startNonce: nonce})
		searchBufIndex = (searchBufIndex + 1) % searchBuffSize

		if len(pending) == searchBuffSize {
			p = &(pending[searchBufIndex])
			cres, _, err := d.queue.EnqueueMapBuffer(d.searchBuffers[p.bufIndex], true,
				cl.MapFlagRead, 0, (1+maxSearchResults)*SIZEOF_UINT32,
				nil)
			if err != nil {
				fmt.Println("Error in Search clEnqueueMapBuffer: ", err)
				return 0, []byte{0}
			}

			results := cres.ByteSlice()
			nfound := binary.LittleEndian.Uint32(results)
			nfound = uint32(math.Min(float64(nfound), float64(maxSearchResults)))
			// OpenCL returns the offsets from the start nonce
			for i := uint32(0); i < nfound; i++ {
				lo := (i + 1) * SIZEOF_UINT32
				hi := (i + 2) * SIZEOF_UINT32
				upperNonce := uint64(binary.LittleEndian.Uint32(results[lo:hi]))
				checkNonce = p.startNonce + upperNonce
				if checkNonce != 0 {
					cn := C.uint64_t(checkNonce)
					ds := C.uint64_t(c.dagSize)
					// We verify that the nonce is indeed a solution by calling Ethash verification function
					// in C on the CPU.
					ret := C.ethash_light_compute_internal(c.ethash.Light.current.ptr, ds, hashToH256(headerHash), cn)
					// TODO: return result first
					if ret.success && h256ToHash(ret.result).Big().Cmp(target256) <= 0 {
						_, err = d.queue.EnqueueUnmapMemObject(d.searchBuffers[p.bufIndex], cres, nil)
						if err != nil {
							fmt.Println("Error in Search clEnqueueUnmapMemObject: ", err)
						}
						if d.openCL12 {
							err = cl.WaitForEvents([]*cl.Event{preReturnEvent})
							if err != nil {
								fmt.Println("Error in Search WaitForEvents: ", err)
							}
						}
						return checkNonce, C.GoBytes(unsafe.Pointer(&ret.mix_hash), C.int(32))
					}

					_, err := d.queue.EnqueueWriteBuffer(d.searchBuffers[p.bufIndex], false, 0, 4, unsafe.Pointer(&zero), nil)
					if err != nil {
						fmt.Println("Error in Search cl: EnqueueWriteBuffer", err)
						return 0, []byte{0}
					}
				}
			}
			_, err = d.queue.EnqueueUnmapMemObject(d.searchBuffers[p.bufIndex], cres, nil)
			if err != nil {
				fmt.Println("Error in Search clEnqueueUnMapMemObject: ", err)
				return 0, []byte{0}
			}
			pending = append(pending[:searchBufIndex], pending[searchBufIndex+1:]...)
		}
	}
	if d.openCL12 {
		err := cl.WaitForEvents([]*cl.Event{preReturnEvent})
		if err != nil {
			fmt.Println("Error in Search clWaitForEvents: ", err)
			return 0, []byte{0}
		}
	}
	return 0, []byte{0}
}

func (c *OpenCLMiner) Verify(block pow.Block) bool {
	return c.ethash.Light.Verify(block)
}
func (c *OpenCLMiner) GetHashrate() int64 {
	// TODO: set in search loop
	return int64(atomic.LoadInt32(&c.hashRate))
}
func (c *OpenCLMiner) Turbo(on bool) {
	// This is GPU mining. Always be turbo.
}

func replaceWords(text string, kvs map[string]string) string {
	for k, v := range kvs {
		text = strings.Replace(text, k, v, -1)
	}
	return text
}

func logErr(err error) {
	if err != nil {
		fmt.Println("Error in OpenCL call:", err)
	}
}

func argErr(err error) error {
	return fmt.Errorf("arg err: %v", err)
}
