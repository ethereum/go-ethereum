package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import "unsafe"

type CommandQueueProperty int

const (
	CommandQueueOutOfOrderExecModeEnable CommandQueueProperty = C.CL_QUEUE_OUT_OF_ORDER_EXEC_MODE_ENABLE
	CommandQueueProfilingEnable          CommandQueueProperty = C.CL_QUEUE_PROFILING_ENABLE
)

type CommandQueue struct {
	clQueue C.cl_command_queue
	device  *Device
}

func releaseCommandQueue(q *CommandQueue) {
	if q.clQueue != nil {
		C.clReleaseCommandQueue(q.clQueue)
		q.clQueue = nil
	}
}

// Call clReleaseCommandQueue on the CommandQueue. Using the CommandQueue after Release will cause a panick.
func (q *CommandQueue) Release() {
	releaseCommandQueue(q)
}

// Blocks until all previously queued OpenCL commands in a command-queue are issued to the associated device and have completed.
func (q *CommandQueue) Finish() error {
	return toError(C.clFinish(q.clQueue))
}

// Issues all previously queued OpenCL commands in a command-queue to the device associated with the command-queue.
func (q *CommandQueue) Flush() error {
	return toError(C.clFlush(q.clQueue))
}

// Enqueues a command to map a region of the buffer object given by buffer into the host address space and returns a pointer to this mapped region.
func (q *CommandQueue) EnqueueMapBuffer(buffer *MemObject, blocking bool, flags MapFlag, offset, size int, eventWaitList []*Event) (*MappedMemObject, *Event, error) {
	var event C.cl_event
	var err C.cl_int
	ptr := C.clEnqueueMapBuffer(q.clQueue, buffer.clMem, clBool(blocking), flags.toCl(), C.size_t(offset), C.size_t(size), C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event, &err)
	if err != C.CL_SUCCESS {
		return nil, nil, toError(err)
	}
	ev := newEvent(event)
	if ptr == nil {
		return nil, ev, ErrUnknown
	}
	return &MappedMemObject{ptr: ptr, size: size}, ev, nil
}

// Enqueues a command to map a region of an image object into the host address space and returns a pointer to this mapped region.
func (q *CommandQueue) EnqueueMapImage(buffer *MemObject, blocking bool, flags MapFlag, origin, region [3]int, eventWaitList []*Event) (*MappedMemObject, *Event, error) {
	cOrigin := sizeT3(origin)
	cRegion := sizeT3(region)
	var event C.cl_event
	var err C.cl_int
	var rowPitch, slicePitch C.size_t
	ptr := C.clEnqueueMapImage(q.clQueue, buffer.clMem, clBool(blocking), flags.toCl(), &cOrigin[0], &cRegion[0], &rowPitch, &slicePitch, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event, &err)
	if err != C.CL_SUCCESS {
		return nil, nil, toError(err)
	}
	ev := newEvent(event)
	if ptr == nil {
		return nil, ev, ErrUnknown
	}
	size := 0 // TODO: could calculate this
	return &MappedMemObject{ptr: ptr, size: size, rowPitch: int(rowPitch), slicePitch: int(slicePitch)}, ev, nil
}

// Enqueues a command to unmap a previously mapped region of a memory object.
func (q *CommandQueue) EnqueueUnmapMemObject(buffer *MemObject, mappedObj *MappedMemObject, eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	if err := C.clEnqueueUnmapMemObject(q.clQueue, buffer.clMem, mappedObj.ptr, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event); err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	return newEvent(event), nil
}

// Enqueues a command to copy a buffer object to another buffer object.
func (q *CommandQueue) EnqueueCopyBuffer(srcBuffer, dstBuffer *MemObject, srcOffset, dstOffset, byteCount int, eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueCopyBuffer(q.clQueue, srcBuffer.clMem, dstBuffer.clMem, C.size_t(srcOffset), C.size_t(dstOffset), C.size_t(byteCount), C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

// Enqueue commands to write to a buffer object from host memory.
func (q *CommandQueue) EnqueueWriteBuffer(buffer *MemObject, blocking bool, offset, dataSize int, dataPtr unsafe.Pointer, eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueWriteBuffer(q.clQueue, buffer.clMem, clBool(blocking), C.size_t(offset), C.size_t(dataSize), dataPtr, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

func (q *CommandQueue) EnqueueWriteBufferFloat32(buffer *MemObject, blocking bool, offset int, data []float32, eventWaitList []*Event) (*Event, error) {
	dataPtr := unsafe.Pointer(&data[0])
	dataSize := int(unsafe.Sizeof(data[0])) * len(data)
	return q.EnqueueWriteBuffer(buffer, blocking, offset, dataSize, dataPtr, eventWaitList)
}

// Enqueue commands to read from a buffer object to host memory.
func (q *CommandQueue) EnqueueReadBuffer(buffer *MemObject, blocking bool, offset, dataSize int, dataPtr unsafe.Pointer, eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueReadBuffer(q.clQueue, buffer.clMem, clBool(blocking), C.size_t(offset), C.size_t(dataSize), dataPtr, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

func (q *CommandQueue) EnqueueReadBufferFloat32(buffer *MemObject, blocking bool, offset int, data []float32, eventWaitList []*Event) (*Event, error) {
	dataPtr := unsafe.Pointer(&data[0])
	dataSize := int(unsafe.Sizeof(data[0])) * len(data)
	return q.EnqueueReadBuffer(buffer, blocking, offset, dataSize, dataPtr, eventWaitList)
}

// Enqueues a command to execute a kernel on a device.
func (q *CommandQueue) EnqueueNDRangeKernel(kernel *Kernel, globalWorkOffset, globalWorkSize, localWorkSize []int, eventWaitList []*Event) (*Event, error) {
	workDim := len(globalWorkSize)
	var globalWorkOffsetList []C.size_t
	var globalWorkOffsetPtr *C.size_t
	if globalWorkOffset != nil {
		globalWorkOffsetList = make([]C.size_t, len(globalWorkOffset))
		for i, off := range globalWorkOffset {
			globalWorkOffsetList[i] = C.size_t(off)
		}
		globalWorkOffsetPtr = &globalWorkOffsetList[0]
	}
	var globalWorkSizeList []C.size_t
	var globalWorkSizePtr *C.size_t
	if globalWorkSize != nil {
		globalWorkSizeList = make([]C.size_t, len(globalWorkSize))
		for i, off := range globalWorkSize {
			globalWorkSizeList[i] = C.size_t(off)
		}
		globalWorkSizePtr = &globalWorkSizeList[0]
	}
	var localWorkSizeList []C.size_t
	var localWorkSizePtr *C.size_t
	if localWorkSize != nil {
		localWorkSizeList = make([]C.size_t, len(localWorkSize))
		for i, off := range localWorkSize {
			localWorkSizeList[i] = C.size_t(off)
		}
		localWorkSizePtr = &localWorkSizeList[0]
	}
	var event C.cl_event
	err := toError(C.clEnqueueNDRangeKernel(q.clQueue, kernel.clKernel, C.cl_uint(workDim), globalWorkOffsetPtr, globalWorkSizePtr, localWorkSizePtr, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

// Enqueues a command to read from a 2D or 3D image object to host memory.
func (q *CommandQueue) EnqueueReadImage(image *MemObject, blocking bool, origin, region [3]int, rowPitch, slicePitch int, data []byte, eventWaitList []*Event) (*Event, error) {
	cOrigin := sizeT3(origin)
	cRegion := sizeT3(region)
	var event C.cl_event
	err := toError(C.clEnqueueReadImage(q.clQueue, image.clMem, clBool(blocking), &cOrigin[0], &cRegion[0], C.size_t(rowPitch), C.size_t(slicePitch), unsafe.Pointer(&data[0]), C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

// Enqueues a command to write from a 2D or 3D image object to host memory.
func (q *CommandQueue) EnqueueWriteImage(image *MemObject, blocking bool, origin, region [3]int, rowPitch, slicePitch int, data []byte, eventWaitList []*Event) (*Event, error) {
	cOrigin := sizeT3(origin)
	cRegion := sizeT3(region)
	var event C.cl_event
	err := toError(C.clEnqueueWriteImage(q.clQueue, image.clMem, clBool(blocking), &cOrigin[0], &cRegion[0], C.size_t(rowPitch), C.size_t(slicePitch), unsafe.Pointer(&data[0]), C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

func (q *CommandQueue) EnqueueFillBuffer(buffer *MemObject, pattern unsafe.Pointer, patternSize, offset, size int, eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueFillBuffer(q.clQueue, buffer.clMem, pattern, C.size_t(patternSize), C.size_t(offset), C.size_t(size), C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

// A synchronization point that enqueues a barrier operation.
func (q *CommandQueue) EnqueueBarrierWithWaitList(eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueBarrierWithWaitList(q.clQueue, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}

// Enqueues a marker command which waits for either a list of events to complete, or all previously enqueued commands to complete.
func (q *CommandQueue) EnqueueMarkerWithWaitList(eventWaitList []*Event) (*Event, error) {
	var event C.cl_event
	err := toError(C.clEnqueueMarkerWithWaitList(q.clQueue, C.cl_uint(len(eventWaitList)), eventListPtr(eventWaitList), &event))
	return newEvent(event), err
}
