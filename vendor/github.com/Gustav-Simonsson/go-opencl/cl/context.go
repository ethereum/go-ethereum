package cl

// #include <stdlib.h>
// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import (
	"runtime"
	"unsafe"
)

const maxImageFormats = 256

type Context struct {
	clContext C.cl_context
	devices   []*Device
}

type MemObject struct {
	clMem C.cl_mem
	size  int
}

func releaseContext(c *Context) {
	if c.clContext != nil {
		C.clReleaseContext(c.clContext)
		c.clContext = nil
	}
}

func releaseMemObject(b *MemObject) {
	if b.clMem != nil {
		C.clReleaseMemObject(b.clMem)
		b.clMem = nil
	}
}

func newMemObject(mo C.cl_mem, size int) *MemObject {
	memObject := &MemObject{clMem: mo, size: size}
	runtime.SetFinalizer(memObject, releaseMemObject)
	return memObject
}

func (b *MemObject) Release() {
	releaseMemObject(b)
}

// TODO: properties
func CreateContext(devices []*Device) (*Context, error) {
	deviceIds := buildDeviceIdList(devices)
	var err C.cl_int
	clContext := C.clCreateContext(nil, C.cl_uint(len(devices)), &deviceIds[0], nil, nil, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	if clContext == nil {
		return nil, ErrUnknown
	}
	context := &Context{clContext: clContext, devices: devices}
	runtime.SetFinalizer(context, releaseContext)
	return context, nil
}

func (ctx *Context) GetSupportedImageFormats(flags MemFlag, imageType MemObjectType) ([]ImageFormat, error) {
	var formats [maxImageFormats]C.cl_image_format
	var nFormats C.cl_uint
	if err := C.clGetSupportedImageFormats(ctx.clContext, C.cl_mem_flags(flags), C.cl_mem_object_type(imageType), maxImageFormats, &formats[0], &nFormats); err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	fmts := make([]ImageFormat, nFormats)
	for i, f := range formats[:nFormats] {
		fmts[i] = ImageFormat{
			ChannelOrder:    ChannelOrder(f.image_channel_order),
			ChannelDataType: ChannelDataType(f.image_channel_data_type),
		}
	}
	return fmts, nil
}

func (ctx *Context) CreateCommandQueue(device *Device, properties CommandQueueProperty) (*CommandQueue, error) {
	var err C.cl_int
	clQueue := C.clCreateCommandQueue(ctx.clContext, device.id, C.cl_command_queue_properties(properties), &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	if clQueue == nil {
		return nil, ErrUnknown
	}
	commandQueue := &CommandQueue{clQueue: clQueue, device: device}
	runtime.SetFinalizer(commandQueue, releaseCommandQueue)
	return commandQueue, nil
}

func (ctx *Context) CreateProgramWithSource(sources []string) (*Program, error) {
	cSources := make([]*C.char, len(sources))
	for i, s := range sources {
		cs := C.CString(s)
		cSources[i] = cs
		defer C.free(unsafe.Pointer(cs))
	}
	var err C.cl_int
	clProgram := C.clCreateProgramWithSource(ctx.clContext, C.cl_uint(len(sources)), &cSources[0], nil, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	if clProgram == nil {
		return nil, ErrUnknown
	}
	program := &Program{clProgram: clProgram, devices: ctx.devices}
	runtime.SetFinalizer(program, releaseProgram)
	return program, nil
}

func (ctx *Context) CreateBufferUnsafe(flags MemFlag, size int, dataPtr unsafe.Pointer) (*MemObject, error) {
	var err C.cl_int
	clBuffer := C.clCreateBuffer(ctx.clContext, C.cl_mem_flags(flags), C.size_t(size), dataPtr, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	if clBuffer == nil {
		return nil, ErrUnknown
	}
	return newMemObject(clBuffer, size), nil
}

func (ctx *Context) CreateEmptyBuffer(flags MemFlag, size int) (*MemObject, error) {
	return ctx.CreateBufferUnsafe(flags, size, nil)
}

func (ctx *Context) CreateEmptyBufferFloat32(flags MemFlag, size int) (*MemObject, error) {
	return ctx.CreateBufferUnsafe(flags, 4*size, nil)
}

func (ctx *Context) CreateBuffer(flags MemFlag, data []byte) (*MemObject, error) {
	return ctx.CreateBufferUnsafe(flags, len(data), unsafe.Pointer(&data[0]))
}

//float64
func (ctx *Context) CreateBufferFloat32(flags MemFlag, data []float32) (*MemObject, error) {
	return ctx.CreateBufferUnsafe(flags, 4*len(data), unsafe.Pointer(&data[0]))
}

func (ctx *Context) CreateUserEvent() (*Event, error) {
	var err C.cl_int
	clEvent := C.clCreateUserEvent(ctx.clContext, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	return newEvent(clEvent), nil
}

func (ctx *Context) Release() {
	releaseContext(ctx)
}

// http://www.khronos.org/registry/cl/sdk/1.2/docs/man/xhtml/clCreateSubBuffer.html
// func (memObject *MemObject) CreateSubBuffer(flags MemFlag, bufferCreateType BufferCreateType, )
