package cl

// #include <stdlib.h>
// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

type BuildError struct {
	Message string
	Device  *Device
}

func (e BuildError) Error() string {
	if e.Device != nil {
		return fmt.Sprintf("cl: build error on %q: %s", e.Device.Name(), e.Message)
	} else {
		return fmt.Sprintf("cl: build error: %s", e.Message)
	}
}

type Program struct {
	clProgram C.cl_program
	devices   []*Device
}

func releaseProgram(p *Program) {
	if p.clProgram != nil {
		C.clReleaseProgram(p.clProgram)
		p.clProgram = nil
	}
}

func (p *Program) Release() {
	releaseProgram(p)
}

func (p *Program) BuildProgram(devices []*Device, options string) error {
	var cOptions *C.char
	if options != "" {
		cOptions = C.CString(options)
		defer C.free(unsafe.Pointer(cOptions))
	}
	var deviceList []C.cl_device_id
	var deviceListPtr *C.cl_device_id
	numDevices := C.cl_uint(len(devices))
	if devices != nil && len(devices) > 0 {
		deviceList = buildDeviceIdList(devices)
		deviceListPtr = &deviceList[0]
	}
	if err := C.clBuildProgram(p.clProgram, numDevices, deviceListPtr, cOptions, nil, nil); err != C.CL_SUCCESS {
		buffer := make([]byte, 4096)
		var bLen C.size_t
		var err C.cl_int

		for _, dev := range p.devices {
			for i := 2; i >= 0; i-- {
				err = C.clGetProgramBuildInfo(p.clProgram, dev.id, C.CL_PROGRAM_BUILD_LOG, C.size_t(len(buffer)), unsafe.Pointer(&buffer[0]), &bLen)
				if err == C.CL_INVALID_VALUE && i > 0 && bLen < 1024*1024 {
					// INVALID_VALUE probably means our buffer isn't large enough
					buffer = make([]byte, bLen)
				} else {
					break
				}
			}
			if err != C.CL_SUCCESS {
				return toError(err)
			}

			if bLen > 1 {
				return BuildError{
					Device:  dev,
					Message: string(buffer[:bLen-1]),
				}
			}
		}

		return BuildError{
			Device:  nil,
			Message: "build failed and produced no log entries",
		}
	}
	return nil
}

func (p *Program) CreateKernel(name string) (*Kernel, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	var err C.cl_int
	clKernel := C.clCreateKernel(p.clProgram, cName, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	kernel := &Kernel{clKernel: clKernel, name: name}
	runtime.SetFinalizer(kernel, releaseKernel)
	return kernel, nil
}
