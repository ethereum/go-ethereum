// Copyright 2014 The go-ethereum Authors
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

// +build opencl

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

/*

  This code have two main entry points:

  1. The initCL(...)  function configures one or more OpenCL device
     (for now only GPU) and loads the Ethash DAG onto device memory

  2. The Search(...) function loads a Ethash nonce into device(s) memory and
     executes the Ethash OpenCL kernel.

  Throughout the code, we refer to "host memory" and "device memory".
  For most systems (e.g. regular PC GPU miner) the host memory is RAM and
  device memory is the GPU global memory (e.g. GDDR5).

  References mentioned in code comments:

  1. https://github.com/ethereum/wiki/wiki/Ethash
  2. https://github.com/ethereum/cpp-ethereum/blob/develop/libethash-cl/ethash_cl_miner.cpp
  3. https://www.khronos.org/registry/cl/sdk/1.2/docs/man/xhtml/
  4. http://amd-dev.wpengine.netdna-cdn.com/wordpress/media/2013/12/AMD_OpenCL_Programming_User_Guide.pdf

*/

type OpenCLDevice struct {
	deviceId int
	device   *cl.Device
	openCL11 bool // OpenCL version 1.1 and 1.2 are handled a bit different
	openCL12 bool

	dagBuf        *cl.MemObject // Ethash full DAG in device mem
	headerBuf     *cl.MemObject // Hash of block-to-mine in device mem
	searchBuffers []*cl.MemObject

	searchKernel *cl.Kernel
	hashKernel   *cl.Kernel

	queue         *cl.CommandQueue
	ctx           *cl.Context
	workGroupSize int

	nonceRand *mrand.Rand // seeded by crypto/rand, see comments where it's initialised
	result    common.Hash
}

type OpenCLMiner struct {
	mu sync.Mutex

	ethash *Ethash // Ethash full DAG & cache in host mem

	deviceIds []int
	devices   []*OpenCLDevice

	dagSize uint64

	hashRate int32 // Go atomics & uint64 have some issues; int32 is supported on all platforms
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
	searchBufSize    = 2
	globalWorkSize   = 1024 * 256
)

func NewCL(deviceIds []int) *OpenCLMiner {
	ids := make([]int, len(deviceIds))
	copy(ids, deviceIds)
	return &OpenCLMiner{
		ethash:    New(),
		dagSize:   0, // to see if we need to update DAG.
		deviceIds: ids,
	}
}

func PrintDevices() {
	fmt.Println("=============================================")
	fmt.Println("============ OpenCL Device Info =============")
	fmt.Println("=============================================")

	var found []*cl.Device

	platforms, err := cl.GetPlatforms()
	if err != nil {
		fmt.Println("Plaform error (check your OpenCL installation):", err)
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
	}
}

// See [2]. We basically do the same here, but the Go OpenCL bindings
// are at a slightly higher abtraction level.
func InitCL(blockNum uint64, c *OpenCLMiner) error {
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
	_ = pow.getDAG(blockNum)     // generates DAG if we don't have it
	pow.Light.getCache(blockNum) // and cache

	c.ethash = pow
	dagSize := uint64(C.ethash_get_datasize(C.uint64_t(blockNum)))
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

	// log warnings but carry on; some device drivers report inaccurate values
	if c.dagSize > devGlobalMem {
		fmt.Printf("WARNING: device memory may be insufficient: %v. DAG size: %v.\n", devGlobalMem, c.dagSize)
	}

	if c.dagSize > devMaxAlloc {
		fmt.Printf("WARNING: DAG size (%v) larger than device max memory allocation size (%v).\n", c.dagSize, devMaxAlloc)
		fmt.Printf("You probably have to export GPU_MAX_ALLOC_PERCENT=95\n")
	}

	fmt.Printf("Initialising device %v: %v\n", deviceId, device.Name())

	context, err := cl.CreateContext([]*cl.Device{device})
	if err != nil {
		return fmt.Errorf("failed creating context: %v", err)
	}

	// TODO: test running with CL_QUEUE_PROFILING_ENABLE for profiling?
	queue, err := context.CreateCommandQueue(device, 0)
	if err != nil {
		return fmt.Errorf("command queue err: %v", err)
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
		return fmt.Errorf("program err: %v", err)
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
		return fmt.Errorf("program build err: %v", err)
	}

	var searchKernelName, hashKernelName string
	searchKernelName = "ethash_search"
	hashKernelName = "ethash_hash"

	searchKernel, err := program.CreateKernel(searchKernelName)
	hashKernel, err := program.CreateKernel(hashKernelName)
	if err != nil {
		return fmt.Errorf("kernel err: %v", err)
	}

	// TODO: when this DAG size appears, patch the Go bindings
	// (context.go) to work with uint64 as size_t
	if c.dagSize > math.MaxInt32 {
		fmt.Println("DAG too large for allocation.")
		return fmt.Errorf("DAG too large for alloc")
	}

	// TODO: patch up Go bindings to work with size_t, will overflow if > maxint32
	// TODO: fuck. shit's gonna overflow around 2017-06-09 12:17:02
	dagBuf := *(new(*cl.MemObject))
	dagBuf, err = context.CreateEmptyBuffer(cl.MemReadOnly, int(c.dagSize))
	if err != nil {
		return fmt.Errorf("allocating dag buf failed: %v", err)
	}

	// write DAG to device mem
	dagPtr := unsafe.Pointer(c.ethash.Full.current.ptr.data)
	_, err = queue.EnqueueWriteBuffer(dagBuf, true, 0, int(c.dagSize), dagPtr, nil)
	if err != nil {
		return fmt.Errorf("writing to dag buf failed: %v", err)
	}

	searchBuffers := make([]*cl.MemObject, searchBufSize)
	for i := 0; i < searchBufSize; i++ {
		searchBuff, err := context.CreateEmptyBuffer(cl.MemWriteOnly, (1+maxSearchResults)*SIZEOF_UINT32)
		if err != nil {
			return fmt.Errorf("search buffer err: %v", err)
		}
		searchBuffers[i] = searchBuff
	}

	headerBuf, err := context.CreateEmptyBuffer(cl.MemReadOnly, 32)
	if err != nil {
		return fmt.Errorf("header buffer err: %v", err)
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

		dagBuf:        dagBuf,
		headerBuf:     headerBuf,
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
	newDagSize := uint64(C.ethash_get_datasize(C.uint64_t(block.NumberU64())))
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
	diff := block.Difficulty()
	target256 := new(big.Int).Div(maxUint256, diff)
	target64 := new(big.Int).Rsh(target256, 192).Uint64()
	var zero uint32 = 0

	d := c.devices[index]

	_, err := d.queue.EnqueueWriteBuffer(d.headerBuf, false, 0, 32, unsafe.Pointer(&headerHash[0]), nil)
	if err != nil {
		fmt.Println("Error in Search clEnqueueWriterBuffer : ", err)
		return 0, []byte{0}
	}

	for i := 0; i < searchBufSize; i++ {
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

	err = d.searchKernel.SetArg(1, d.headerBuf)
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}

	err = d.searchKernel.SetArg(2, d.dagBuf)
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}

	err = d.searchKernel.SetArg(4, target64)
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}
	err = d.searchKernel.SetArg(5, uint32(math.MaxUint32))
	if err != nil {
		fmt.Println("Error in Search clSetKernelArg : ", err)
		return 0, []byte{0}
	}

	// wait on this before returning
	var preReturnEvent *cl.Event
	if d.openCL12 {
		preReturnEvent, err = d.ctx.CreateUserEvent()
		if err != nil {
			fmt.Println("Error in Search create CL user event : ", err)
			return 0, []byte{0}
		}
	}

	pending := make([]pendingSearch, 0, searchBufSize)
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
			// TODO: verify if this is correct hash rate calculation
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
		err = d.searchKernel.SetArg(3, nonce)
		if err != nil {
			fmt.Println("Error in Search clSetKernelArg : ", err)
			return 0, []byte{0}
		}

		// execute kernel
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
		searchBufIndex = (searchBufIndex + 1) % searchBufSize

		if len(pending) == searchBufSize {
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
					// We verify that the nonce is indeed a solution by
					// executing the Ethash verification function (on the CPU).
					cache := c.ethash.Light.getCache(block.NumberU64())
					ok, mixDigest, result := cache.compute(c.dagSize, headerHash, checkNonce)

					// TODO: return result first
					if ok && result.Big().Cmp(target256) <= 0 {
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
						return checkNonce, mixDigest.Bytes()
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
