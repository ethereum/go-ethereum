package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import (
	"fmt"
	"unsafe"
)

type ErrUnsupportedArgumentType struct {
	Index int
	Value interface{}
}

func (e ErrUnsupportedArgumentType) Error() string {
	return fmt.Sprintf("cl: unsupported argument type for index %d: %+v", e.Index, e.Value)
}

type Kernel struct {
	clKernel C.cl_kernel
	name     string
}

type LocalBuffer int

func releaseKernel(k *Kernel) {
	if k.clKernel != nil {
		C.clReleaseKernel(k.clKernel)
		k.clKernel = nil
	}
}

func (k *Kernel) Release() {
	releaseKernel(k)
}

func (k *Kernel) SetArgs(args ...interface{}) error {
	for index, arg := range args {
		if err := k.SetArg(index, arg); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kernel) SetArg(index int, arg interface{}) error {
	switch val := arg.(type) {
	case uint8:
		return k.SetArgUint8(index, val)
	case int8:
		return k.SetArgInt8(index, val)
	case uint32:
		return k.SetArgUint32(index, val)
	case uint64:
		return k.SetArgUint64(index, val)
	case int32:
		return k.SetArgInt32(index, val)
	case float32:
		return k.SetArgFloat32(index, val)
	case *MemObject:
		return k.SetArgBuffer(index, val)
	case LocalBuffer:
		return k.SetArgLocal(index, int(val))
	default:
		return ErrUnsupportedArgumentType{Index: index, Value: arg}
	}
}

func (k *Kernel) SetArgBuffer(index int, buffer *MemObject) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(buffer.clMem)), unsafe.Pointer(&buffer.clMem))
}

func (k *Kernel) SetArgFloat32(index int, val float32) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgInt8(index int, val int8) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgUint8(index int, val uint8) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgInt32(index int, val int32) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgUint32(index int, val uint32) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgUint64(index int, val uint64) error {
	return k.SetArgUnsafe(index, int(unsafe.Sizeof(val)), unsafe.Pointer(&val))
}

func (k *Kernel) SetArgLocal(index int, size int) error {
	return k.SetArgUnsafe(index, size, nil)
}

func (k *Kernel) SetArgUnsafe(index, argSize int, arg unsafe.Pointer) error {
	//fmt.Println("FUNKY: ", index, argSize)
	return toError(C.clSetKernelArg(k.clKernel, C.cl_uint(index), C.size_t(argSize), arg))
}

func (k *Kernel) PreferredWorkGroupSizeMultiple(device *Device) (int, error) {
	var size C.size_t
	err := C.clGetKernelWorkGroupInfo(k.clKernel, device.nullableId(), C.CL_KERNEL_PREFERRED_WORK_GROUP_SIZE_MULTIPLE, C.size_t(unsafe.Sizeof(size)), unsafe.Pointer(&size), nil)
	return int(size), toError(err)
}

func (k *Kernel) WorkGroupSize(device *Device) (int, error) {
	var size C.size_t
	err := C.clGetKernelWorkGroupInfo(k.clKernel, device.nullableId(), C.CL_KERNEL_WORK_GROUP_SIZE, C.size_t(unsafe.Sizeof(size)), unsafe.Pointer(&size), nil)
	return int(size), toError(err)
}

func (k *Kernel) NumArgs() (int, error) {
	var num C.cl_uint
	err := C.clGetKernelInfo(k.clKernel, C.CL_KERNEL_NUM_ARGS, C.size_t(unsafe.Sizeof(num)), unsafe.Pointer(&num), nil)
	return int(num), toError(err)
}
