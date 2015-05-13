package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"text/template"
	"unsafe"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/samuel/go-opencl/cl"
)

//#include <stdint.h>
import "C"

type params struct {
	GroupSize  int
	DagSize    int
	Accesses   int
	MaxOutputs int
}

const (
	kernelFn         = "ethash_kernel.cl"
	searchBuffSize   = 2
	maxSearchResults = 63
	searchBatchSize  = 1024
	dagSize          = 1024 * 32

	SIZEOF_UINT32    = 4
	ETHASH_MIX_BYTES = 128
	ACCESSES         = 64
)

var (
	workGroupSize = 256
	pow           *ethash.Ethash
)

type EthashCL struct {
	// bufffers
	dagBuff       *cl.MemObject
	headerBuff    *cl.MemObject
	searchBuffers []*cl.MemObject

	// kernels
	searchKernel *cl.Kernel
	hashKernelel *cl.Kernel

	// contexts
	queue         *cl.CommandQueue
	ctx           *cl.Context
	device        *cl.Device
	workGroupSize int

	// resurts
	hash common.Hash
}

func New() (*EthashCL, error) {
	platforms, err := cl.GetPlatforms()
	if err != nil {
		return nil, fmt.Errorf("platform err:", err)
	}

	platform := platforms[0]
	fmt.Println("using platform:", platform.Name())

	devices, err := cl.GetDevices(platform, cl.DeviceTypeGPU)
	if err != nil {
		return nil, fmt.Errorf("device err:", err)
	}

	device := devices[0]
	fmt.Println("using device:", device.Name())

	if device.Version() == "OpenCL 1.0" {
		return nil, fmt.Errorf(device.Version(), "not supported")
	}

	context, err := cl.CreateContext([]*cl.Device{device})
	if err != nil {
		return nil, fmt.Errorf("failed creating context:", err)
	}

	queue, err := context.CreateCommandQueue(device, 0)
	if err != nil {
		return nil, fmt.Errorf("command queue err:", err)
	}

	fc, err := ioutil.ReadFile(kernelFn)
	if err != nil {
		return nil, fmt.Errorf("reading opencl.go err:", err)
	}

	tmpl, err := template.New(kernelFn).Parse(string(fc))
	if err != nil {
		return nil, fmt.Errorf("template err:", err)
	}

	workGroupSize = ((workGroupSize + 7) / 8) * 8

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, params{
		GroupSize: workGroupSize, Accesses: ACCESSES, MaxOutputs: 4, DagSize: dagSize / ETHASH_MIX_BYTES,
	})
	if err != nil {
		return nil, fmt.Errorf("template err:", err)
	}
	source, err := ioutil.ReadAll(&buffer)
	if err != nil {
		return nil, fmt.Errorf("buffer err:", err)
	}

	program, err := context.CreateProgramWithSource([]string{string(source)})
	if err != nil {
		return nil, fmt.Errorf("program err:", err)
	}

	err = program.BuildProgram([]*cl.Device{device}, "")
	if err != nil {
		return nil, fmt.Errorf("program build err:", err)
	}

	searchKernel, err := program.CreateKernel("ethash_search")
	if err != nil {
		return nil, fmt.Errorf("kernel err:", err)
	}

	dagBuff, err := context.CreateEmptyBuffer(cl.MemReadOnly, dagSize)
	if err != nil {
		return nil, fmt.Errorf("dag buffer err:", err)
	}

	pow, err = ethash.NewForTesting()
	if err != nil {
		return nil, fmt.Errorf("dag err:", err)
	}
	dag := pow.Full.DAG(0).Ptr()
	fmt.Println("DAG ptr", dag)

	queue.EnqueueWriteBuffer(dagBuff, true, 0, dagSize, dag, nil)

	headerBuff, err := context.CreateEmptyBuffer(cl.MemReadOnly, 32)
	if err != nil {
		return nil, fmt.Errorf("header buffer err:", err)
	}

	searchBuffers := make([]*cl.MemObject, searchBuffSize)
	for i := 0; i < searchBuffSize; i++ {
		searchBuff, err := context.CreateEmptyBuffer(cl.MemWriteOnly, maxSearchResults+1*SIZEOF_UINT32)
		if err != nil {
			return nil, fmt.Errorf("search buffer err:", err)
		}
		searchBuffers[i] = searchBuff
	}

	return &EthashCL{
		ctx:           context,
		device:        device,
		dagBuff:       dagBuff,
		searchBuffers: searchBuffers,
		headerBuff:    headerBuff,
		searchKernel:  searchKernel,
		queue:         queue,
		workGroupSize: workGroupSize,
	}, nil
}

func argErr(err error) error {
	return fmt.Errorf("arg err: %v", err)
}

func (h *EthashCL) Search(header common.Hash, target uint64, doneFn func([]uint64) bool) error {
	var zero uint32 = 0

	for i := 0; i < searchBuffSize; i++ {
		h.queue.EnqueueWriteBuffer(h.searchBuffers[i], false, 0, 4, unsafe.Pointer(&zero), nil)
	}
	// wait for all search buffers to complete
	h.queue.Finish()

	err := h.searchKernel.SetArg(1, h.headerBuff)
	if err != nil {
		return argErr(err)
	}
	err = h.searchKernel.SetArg(2, h.dagBuff)
	if err != nil {
		return argErr(err)
	}
	err = h.searchKernel.SetArg(4, target)
	if err != nil {
		return argErr(err)
	}
	err = h.searchKernel.SetArg(5, uint64(math.MaxUint64))
	if err != nil {
		return argErr(err)
	}

	// TODO make multi buffer
	const buff = 0
	for nonce := uint64(10); ; nonce += searchBatchSize {
		err = h.searchKernel.SetArg(0, h.searchBuffers[buff])
		if err != nil {
			return argErr(err)
		}
		err = h.searchKernel.SetArg(3, nonce)
		if err != nil {
			return argErr(err)
		}

		_, err := h.queue.EnqueueNDRangeKernel(
			h.searchKernel,
			[]int{0},
			[]int{searchBatchSize},
			[]int{h.workGroupSize},
			nil)
		if err != nil {
			return fmt.Errorf("exec err: %v", err)
		}

		cres, _, err := h.queue.EnqueueMapBuffer(h.searchBuffers[buff], true, cl.MapFlagRead, 0, (1+maxSearchResults)*SIZEOF_UINT32, nil)
		if err != nil {
			return fmt.Errorf("buffer mapping err: %v", err)
		}
		h.queue.Flush()

		results := cres.ByteSlice()
		nfound := binary.BigEndian.Uint32(results)
		fmt.Println("sollutions found:", nfound)
		nfound = uint32(math.Min(float64(nfound), float64(maxSearchResults)))

		nonces := make([]uint64, maxSearchResults)
		for i := uint32(0); i < nfound; i++ {
			nonces[i] = nonce + uint64(binary.BigEndian.Uint32(results[1+i*SIZEOF_UINT32:]))
		}
		h.queue.EnqueueUnmapMemObject(h.searchBuffers[buff], cres, nil)
		if doneFn(nonces) {
			break
		}

	}

	return nil
}

func rndHash() common.Hash {
	var h common.Hash
	rand.Read(h[:])
	return h
}

func main() {
	glog.SetV(6)
	glog.SetToStderr(true)

	fmt.Println("initialising OpenCL miner...")

	gpu, err := New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("OpenCL miner initialised")

	hash := rndHash()
	fmt.Printf("Searching for solution for %x\n", hash)
	err = gpu.Search(hash, 0x000000ffffffffff, func(res []uint64) bool {
		fmt.Printf("found: %x\n", res)
		return true
	})
	if err != nil {
		fmt.Println("search err:", err)
		os.Exit(1)
	}
	gpu.ctx.Release()
	gpu.dagBuff.Release()
	gpu.headerBuff.Release()
	gpu.searchKernel.Release()
	gpu.queue.Release()
	for _, searchBuffer := range gpu.searchBuffers {
		searchBuffer.Release()
	}
}
