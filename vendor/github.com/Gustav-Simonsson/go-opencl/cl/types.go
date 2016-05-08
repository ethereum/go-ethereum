package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

var (
	ErrUnknown = errors.New("cl: unknown error") // Generally an unexpected result from an OpenCL function (e.g. CL_SUCCESS but null pointer)
)

type ErrOther int

func (e ErrOther) Error() string {
	return fmt.Sprintf("cl: error %d", int(e))
}

var (
	ErrDeviceNotFound                     = errors.New("cl: Device Not Found")
	ErrDeviceNotAvailable                 = errors.New("cl: Device Not Available")
	ErrCompilerNotAvailable               = errors.New("cl: Compiler Not Available")
	ErrMemObjectAllocationFailure         = errors.New("cl: Mem Object Allocation Failure")
	ErrOutOfResources                     = errors.New("cl: Out Of Resources")
	ErrOutOfHostMemory                    = errors.New("cl: Out Of Host Memory")
	ErrProfilingInfoNotAvailable          = errors.New("cl: Profiling Info Not Available")
	ErrMemCopyOverlap                     = errors.New("cl: Mem Copy Overlap")
	ErrImageFormatMismatch                = errors.New("cl: Image Format Mismatch")
	ErrImageFormatNotSupported            = errors.New("cl: Image Format Not Supported")
	ErrBuildProgramFailure                = errors.New("cl: Build Program Failure")
	ErrMapFailure                         = errors.New("cl: Map Failure")
	ErrMisalignedSubBufferOffset          = errors.New("cl: Misaligned Sub Buffer Offset")
	ErrExecStatusErrorForEventsInWaitList = errors.New("cl: Exec Status Error For Events In Wait List")
	ErrCompileProgramFailure              = errors.New("cl: Compile Program Failure")
	ErrLinkerNotAvailable                 = errors.New("cl: Linker Not Available")
	ErrLinkProgramFailure                 = errors.New("cl: Link Program Failure")
	ErrDevicePartitionFailed              = errors.New("cl: Device Partition Failed")
	ErrKernelArgInfoNotAvailable          = errors.New("cl: Kernel Arg Info Not Available")
	ErrInvalidValue                       = errors.New("cl: Invalid Value")
	ErrInvalidDeviceType                  = errors.New("cl: Invalid Device Type")
	ErrInvalidPlatform                    = errors.New("cl: Invalid Platform")
	ErrInvalidDevice                      = errors.New("cl: Invalid Device")
	ErrInvalidContext                     = errors.New("cl: Invalid Context")
	ErrInvalidQueueProperties             = errors.New("cl: Invalid Queue Properties")
	ErrInvalidCommandQueue                = errors.New("cl: Invalid Command Queue")
	ErrInvalidHostPtr                     = errors.New("cl: Invalid Host Ptr")
	ErrInvalidMemObject                   = errors.New("cl: Invalid Mem Object")
	ErrInvalidImageFormatDescriptor       = errors.New("cl: Invalid Image Format Descriptor")
	ErrInvalidImageSize                   = errors.New("cl: Invalid Image Size")
	ErrInvalidSampler                     = errors.New("cl: Invalid Sampler")
	ErrInvalidBinary                      = errors.New("cl: Invalid Binary")
	ErrInvalidBuildOptions                = errors.New("cl: Invalid Build Options")
	ErrInvalidProgram                     = errors.New("cl: Invalid Program")
	ErrInvalidProgramExecutable           = errors.New("cl: Invalid Program Executable")
	ErrInvalidKernelName                  = errors.New("cl: Invalid Kernel Name")
	ErrInvalidKernelDefinition            = errors.New("cl: Invalid Kernel Definition")
	ErrInvalidKernel                      = errors.New("cl: Invalid Kernel")
	ErrInvalidArgIndex                    = errors.New("cl: Invalid Arg Index")
	ErrInvalidArgValue                    = errors.New("cl: Invalid Arg Value")
	ErrInvalidArgSize                     = errors.New("cl: Invalid Arg Size")
	ErrInvalidKernelArgs                  = errors.New("cl: Invalid Kernel Args")
	ErrInvalidWorkDimension               = errors.New("cl: Invalid Work Dimension")
	ErrInvalidWorkGroupSize               = errors.New("cl: Invalid Work Group Size")
	ErrInvalidWorkItemSize                = errors.New("cl: Invalid Work Item Size")
	ErrInvalidGlobalOffset                = errors.New("cl: Invalid Global Offset")
	ErrInvalidEventWaitList               = errors.New("cl: Invalid Event Wait List")
	ErrInvalidEvent                       = errors.New("cl: Invalid Event")
	ErrInvalidOperation                   = errors.New("cl: Invalid Operation")
	ErrInvalidGlObject                    = errors.New("cl: Invalid Gl Object")
	ErrInvalidBufferSize                  = errors.New("cl: Invalid Buffer Size")
	ErrInvalidMipLevel                    = errors.New("cl: Invalid Mip Level")
	ErrInvalidGlobalWorkSize              = errors.New("cl: Invalid Global Work Size")
	ErrInvalidProperty                    = errors.New("cl: Invalid Property")
	ErrInvalidImageDescriptor             = errors.New("cl: Invalid Image Descriptor")
	ErrInvalidCompilerOptions             = errors.New("cl: Invalid Compiler Options")
	ErrInvalidLinkerOptions               = errors.New("cl: Invalid Linker Options")
	ErrInvalidDevicePartitionCount        = errors.New("cl: Invalid Device Partition Count")
)
var errorMap = map[C.cl_int]error{
	C.CL_SUCCESS:                                   nil,
	C.CL_DEVICE_NOT_FOUND:                          ErrDeviceNotFound,
	C.CL_DEVICE_NOT_AVAILABLE:                      ErrDeviceNotAvailable,
	C.CL_COMPILER_NOT_AVAILABLE:                    ErrCompilerNotAvailable,
	C.CL_MEM_OBJECT_ALLOCATION_FAILURE:             ErrMemObjectAllocationFailure,
	C.CL_OUT_OF_RESOURCES:                          ErrOutOfResources,
	C.CL_OUT_OF_HOST_MEMORY:                        ErrOutOfHostMemory,
	C.CL_PROFILING_INFO_NOT_AVAILABLE:              ErrProfilingInfoNotAvailable,
	C.CL_MEM_COPY_OVERLAP:                          ErrMemCopyOverlap,
	C.CL_IMAGE_FORMAT_MISMATCH:                     ErrImageFormatMismatch,
	C.CL_IMAGE_FORMAT_NOT_SUPPORTED:                ErrImageFormatNotSupported,
	C.CL_BUILD_PROGRAM_FAILURE:                     ErrBuildProgramFailure,
	C.CL_MAP_FAILURE:                               ErrMapFailure,
	C.CL_MISALIGNED_SUB_BUFFER_OFFSET:              ErrMisalignedSubBufferOffset,
	C.CL_EXEC_STATUS_ERROR_FOR_EVENTS_IN_WAIT_LIST: ErrExecStatusErrorForEventsInWaitList,
	C.CL_INVALID_VALUE:                             ErrInvalidValue,
	C.CL_INVALID_DEVICE_TYPE:                       ErrInvalidDeviceType,
	C.CL_INVALID_PLATFORM:                          ErrInvalidPlatform,
	C.CL_INVALID_DEVICE:                            ErrInvalidDevice,
	C.CL_INVALID_CONTEXT:                           ErrInvalidContext,
	C.CL_INVALID_QUEUE_PROPERTIES:                  ErrInvalidQueueProperties,
	C.CL_INVALID_COMMAND_QUEUE:                     ErrInvalidCommandQueue,
	C.CL_INVALID_HOST_PTR:                          ErrInvalidHostPtr,
	C.CL_INVALID_MEM_OBJECT:                        ErrInvalidMemObject,
	C.CL_INVALID_IMAGE_FORMAT_DESCRIPTOR:           ErrInvalidImageFormatDescriptor,
	C.CL_INVALID_IMAGE_SIZE:                        ErrInvalidImageSize,
	C.CL_INVALID_SAMPLER:                           ErrInvalidSampler,
	C.CL_INVALID_BINARY:                            ErrInvalidBinary,
	C.CL_INVALID_BUILD_OPTIONS:                     ErrInvalidBuildOptions,
	C.CL_INVALID_PROGRAM:                           ErrInvalidProgram,
	C.CL_INVALID_PROGRAM_EXECUTABLE:                ErrInvalidProgramExecutable,
	C.CL_INVALID_KERNEL_NAME:                       ErrInvalidKernelName,
	C.CL_INVALID_KERNEL_DEFINITION:                 ErrInvalidKernelDefinition,
	C.CL_INVALID_KERNEL:                            ErrInvalidKernel,
	C.CL_INVALID_ARG_INDEX:                         ErrInvalidArgIndex,
	C.CL_INVALID_ARG_VALUE:                         ErrInvalidArgValue,
	C.CL_INVALID_ARG_SIZE:                          ErrInvalidArgSize,
	C.CL_INVALID_KERNEL_ARGS:                       ErrInvalidKernelArgs,
	C.CL_INVALID_WORK_DIMENSION:                    ErrInvalidWorkDimension,
	C.CL_INVALID_WORK_GROUP_SIZE:                   ErrInvalidWorkGroupSize,
	C.CL_INVALID_WORK_ITEM_SIZE:                    ErrInvalidWorkItemSize,
	C.CL_INVALID_GLOBAL_OFFSET:                     ErrInvalidGlobalOffset,
	C.CL_INVALID_EVENT_WAIT_LIST:                   ErrInvalidEventWaitList,
	C.CL_INVALID_EVENT:                             ErrInvalidEvent,
	C.CL_INVALID_OPERATION:                         ErrInvalidOperation,
	C.CL_INVALID_GL_OBJECT:                         ErrInvalidGlObject,
	C.CL_INVALID_BUFFER_SIZE:                       ErrInvalidBufferSize,
	C.CL_INVALID_MIP_LEVEL:                         ErrInvalidMipLevel,
	C.CL_INVALID_GLOBAL_WORK_SIZE:                  ErrInvalidGlobalWorkSize,
	C.CL_INVALID_PROPERTY:                          ErrInvalidProperty,
}

func toError(code C.cl_int) error {
	if err, ok := errorMap[code]; ok {
		return err
	}
	return ErrOther(code)
}

type LocalMemType int

const (
	LocalMemTypeNone   LocalMemType = C.CL_NONE
	LocalMemTypeGlobal LocalMemType = C.CL_GLOBAL
	LocalMemTypeLocal  LocalMemType = C.CL_LOCAL
)

var localMemTypeMap = map[LocalMemType]string{
	LocalMemTypeNone:   "None",
	LocalMemTypeGlobal: "Global",
	LocalMemTypeLocal:  "Local",
}

func (t LocalMemType) String() string {
	name := localMemTypeMap[t]
	if name == "" {
		name = "Unknown"
	}
	return name
}

type ExecCapability int

const (
	ExecCapabilityKernel       ExecCapability = C.CL_EXEC_KERNEL        // The OpenCL device can execute OpenCL kernels.
	ExecCapabilityNativeKernel ExecCapability = C.CL_EXEC_NATIVE_KERNEL // The OpenCL device can execute native kernels.
)

func (ec ExecCapability) String() string {
	var parts []string
	if ec&ExecCapabilityKernel != 0 {
		parts = append(parts, "Kernel")
	}
	if ec&ExecCapabilityNativeKernel != 0 {
		parts = append(parts, "NativeKernel")
	}
	if parts == nil {
		return ""
	}
	return strings.Join(parts, "|")
}

type MemCacheType int

const (
	MemCacheTypeNone           MemCacheType = C.CL_NONE
	MemCacheTypeReadOnlyCache  MemCacheType = C.CL_READ_ONLY_CACHE
	MemCacheTypeReadWriteCache MemCacheType = C.CL_READ_WRITE_CACHE
)

func (ct MemCacheType) String() string {
	switch ct {
	case MemCacheTypeNone:
		return "None"
	case MemCacheTypeReadOnlyCache:
		return "ReadOnly"
	case MemCacheTypeReadWriteCache:
		return "ReadWrite"
	}
	return fmt.Sprintf("Unknown(%x)", int(ct))
}

type MemFlag int

const (
	MemReadWrite    MemFlag = C.CL_MEM_READ_WRITE
	MemWriteOnly    MemFlag = C.CL_MEM_WRITE_ONLY
	MemReadOnly     MemFlag = C.CL_MEM_READ_ONLY
	MemUseHostPtr   MemFlag = C.CL_MEM_USE_HOST_PTR
	MemAllocHostPtr MemFlag = C.CL_MEM_ALLOC_HOST_PTR
	MemCopyHostPtr  MemFlag = C.CL_MEM_COPY_HOST_PTR

	MemWriteOnlyHost MemFlag = C.CL_MEM_HOST_WRITE_ONLY
	MemReadOnlyHost  MemFlag = C.CL_MEM_HOST_READ_ONLY
	MemNoAccessHost  MemFlag = C.CL_MEM_HOST_NO_ACCESS
)

type MemObjectType int

const (
	MemObjectTypeBuffer  MemObjectType = C.CL_MEM_OBJECT_BUFFER
	MemObjectTypeImage2D MemObjectType = C.CL_MEM_OBJECT_IMAGE2D
	MemObjectTypeImage3D MemObjectType = C.CL_MEM_OBJECT_IMAGE3D
)

type MapFlag int

const (
	// This flag specifies that the region being mapped in the memory object is being mapped for reading.
	MapFlagRead                  MapFlag = C.CL_MAP_READ
	MapFlagWrite                 MapFlag = C.CL_MAP_WRITE
	MapFlagWriteInvalidateRegion MapFlag = C.CL_MAP_WRITE_INVALIDATE_REGION
)

func (mf MapFlag) toCl() C.cl_map_flags {
	return C.cl_map_flags(mf)
}

type ChannelOrder int

const (
	ChannelOrderR         ChannelOrder = C.CL_R
	ChannelOrderA         ChannelOrder = C.CL_A
	ChannelOrderRG        ChannelOrder = C.CL_RG
	ChannelOrderRA        ChannelOrder = C.CL_RA
	ChannelOrderRGB       ChannelOrder = C.CL_RGB
	ChannelOrderRGBA      ChannelOrder = C.CL_RGBA
	ChannelOrderBGRA      ChannelOrder = C.CL_BGRA
	ChannelOrderARGB      ChannelOrder = C.CL_ARGB
	ChannelOrderIntensity ChannelOrder = C.CL_INTENSITY
	ChannelOrderLuminance ChannelOrder = C.CL_LUMINANCE
	ChannelOrderRx        ChannelOrder = C.CL_Rx
	ChannelOrderRGx       ChannelOrder = C.CL_RGx
	ChannelOrderRGBx      ChannelOrder = C.CL_RGBx
)

var channelOrderNameMap = map[ChannelOrder]string{
	ChannelOrderR:         "R",
	ChannelOrderA:         "A",
	ChannelOrderRG:        "RG",
	ChannelOrderRA:        "RA",
	ChannelOrderRGB:       "RGB",
	ChannelOrderRGBA:      "RGBA",
	ChannelOrderBGRA:      "BGRA",
	ChannelOrderARGB:      "ARGB",
	ChannelOrderIntensity: "Intensity",
	ChannelOrderLuminance: "Luminance",
	ChannelOrderRx:        "Rx",
	ChannelOrderRGx:       "RGx",
	ChannelOrderRGBx:      "RGBx",
}

func (co ChannelOrder) String() string {
	name := channelOrderNameMap[co]
	if name == "" {
		name = fmt.Sprintf("Unknown(%x)", int(co))
	}
	return name
}

type ChannelDataType int

const (
	ChannelDataTypeSNormInt8      ChannelDataType = C.CL_SNORM_INT8
	ChannelDataTypeSNormInt16     ChannelDataType = C.CL_SNORM_INT16
	ChannelDataTypeUNormInt8      ChannelDataType = C.CL_UNORM_INT8
	ChannelDataTypeUNormInt16     ChannelDataType = C.CL_UNORM_INT16
	ChannelDataTypeUNormShort565  ChannelDataType = C.CL_UNORM_SHORT_565
	ChannelDataTypeUNormShort555  ChannelDataType = C.CL_UNORM_SHORT_555
	ChannelDataTypeUNormInt101010 ChannelDataType = C.CL_UNORM_INT_101010
	ChannelDataTypeSignedInt8     ChannelDataType = C.CL_SIGNED_INT8
	ChannelDataTypeSignedInt16    ChannelDataType = C.CL_SIGNED_INT16
	ChannelDataTypeSignedInt32    ChannelDataType = C.CL_SIGNED_INT32
	ChannelDataTypeUnsignedInt8   ChannelDataType = C.CL_UNSIGNED_INT8
	ChannelDataTypeUnsignedInt16  ChannelDataType = C.CL_UNSIGNED_INT16
	ChannelDataTypeUnsignedInt32  ChannelDataType = C.CL_UNSIGNED_INT32
	ChannelDataTypeHalfFloat      ChannelDataType = C.CL_HALF_FLOAT
	ChannelDataTypeFloat          ChannelDataType = C.CL_FLOAT
)

var channelDataTypeNameMap = map[ChannelDataType]string{
	ChannelDataTypeSNormInt8:      "SNormInt8",
	ChannelDataTypeSNormInt16:     "SNormInt16",
	ChannelDataTypeUNormInt8:      "UNormInt8",
	ChannelDataTypeUNormInt16:     "UNormInt16",
	ChannelDataTypeUNormShort565:  "UNormShort565",
	ChannelDataTypeUNormShort555:  "UNormShort555",
	ChannelDataTypeUNormInt101010: "UNormInt101010",
	ChannelDataTypeSignedInt8:     "SignedInt8",
	ChannelDataTypeSignedInt16:    "SignedInt16",
	ChannelDataTypeSignedInt32:    "SignedInt32",
	ChannelDataTypeUnsignedInt8:   "UnsignedInt8",
	ChannelDataTypeUnsignedInt16:  "UnsignedInt16",
	ChannelDataTypeUnsignedInt32:  "UnsignedInt32",
	ChannelDataTypeHalfFloat:      "HalfFloat",
	ChannelDataTypeFloat:          "Float",
}

func (ct ChannelDataType) String() string {
	name := channelDataTypeNameMap[ct]
	if name == "" {
		name = fmt.Sprintf("Unknown(%x)", int(ct))
	}
	return name
}

type ImageFormat struct {
	ChannelOrder    ChannelOrder
	ChannelDataType ChannelDataType
}

func (f ImageFormat) toCl() C.cl_image_format {
	var format C.cl_image_format
	format.image_channel_order = C.cl_channel_order(f.ChannelOrder)
	format.image_channel_data_type = C.cl_channel_type(f.ChannelDataType)
	return format
}

type ProfilingInfo int

const (
	// A 64-bit value that describes the current device time counter in
	// nanoseconds when the command identified by event is enqueued in
	// a command-queue by the host.
	ProfilingInfoCommandQueued ProfilingInfo = C.CL_PROFILING_COMMAND_QUEUED
	// A 64-bit value that describes the current device time counter in
	// nanoseconds when the command identified by event that has been
	// enqueued is submitted by the host to the device associated with the command-queue.
	ProfilingInfoCommandSubmit ProfilingInfo = C.CL_PROFILING_COMMAND_SUBMIT
	// A 64-bit value that describes the current device time counter in
	// nanoseconds when the command identified by event starts execution on the device.
	ProfilingInfoCommandStart ProfilingInfo = C.CL_PROFILING_COMMAND_START
	// A 64-bit value that describes the current device time counter in
	// nanoseconds when the command identified by event has finished
	// execution on the device.
	ProfilingInfoCommandEnd ProfilingInfo = C.CL_PROFILING_COMMAND_END
)

type CommmandExecStatus int

const (
	CommmandExecStatusComplete  CommmandExecStatus = C.CL_COMPLETE
	CommmandExecStatusRunning   CommmandExecStatus = C.CL_RUNNING
	CommmandExecStatusSubmitted CommmandExecStatus = C.CL_SUBMITTED
	CommmandExecStatusQueued    CommmandExecStatus = C.CL_QUEUED
)

type Event struct {
	clEvent C.cl_event
}

func releaseEvent(ev *Event) {
	if ev.clEvent != nil {
		C.clReleaseEvent(ev.clEvent)
		ev.clEvent = nil
	}
}

func (e *Event) Release() {
	releaseEvent(e)
}

func (e *Event) GetEventProfilingInfo(paramName ProfilingInfo) (int64, error) {
	var paramValue C.cl_ulong
	if err := C.clGetEventProfilingInfo(e.clEvent, C.cl_profiling_info(paramName), C.size_t(unsafe.Sizeof(paramValue)), unsafe.Pointer(&paramValue), nil); err != C.CL_SUCCESS {
		return 0, toError(err)
	}
	return int64(paramValue), nil
}

// Sets the execution status of a user event object.
//
// `status` specifies the new execution status to be set and
// can be CL_COMPLETE or a negative integer value to indicate
// an error. A negative integer value causes all enqueued commands
// that wait on this user event to be terminated. clSetUserEventStatus
// can only be called once to change the execution status of event.
func (e *Event) SetUserEventStatus(status int) error {
	return toError(C.clSetUserEventStatus(e.clEvent, C.cl_int(status)))
}

// Waits on the host thread for commands identified by event objects in
// events to complete. A command is considered complete if its execution
// status is CL_COMPLETE or a negative value. The events specified in
// event_list act as synchronization points.
//
// If the cl_khr_gl_event extension is enabled, event objects can also be
// used to reflect the status of an OpenGL sync object. The sync object
// in turn refers to a fence command executing in an OpenGL command
// stream. This provides another method of coordinating sharing of buffers
// and images between OpenGL and OpenCL.
func WaitForEvents(events []*Event) error {
	return toError(C.clWaitForEvents(C.cl_uint(len(events)), eventListPtr(events)))
}

func newEvent(clEvent C.cl_event) *Event {
	ev := &Event{clEvent: clEvent}
	runtime.SetFinalizer(ev, releaseEvent)
	return ev
}

func eventListPtr(el []*Event) *C.cl_event {
	if el == nil {
		return nil
	}
	elist := make([]C.cl_event, len(el))
	for i, e := range el {
		elist[i] = e.clEvent
	}
	return (*C.cl_event)(&elist[0])
}

func clBool(b bool) C.cl_bool {
	if b {
		return C.CL_TRUE
	}
	return C.CL_FALSE
}

func sizeT3(i3 [3]int) [3]C.size_t {
	var val [3]C.size_t
	val[0] = C.size_t(i3[0])
	val[1] = C.size_t(i3[1])
	val[2] = C.size_t(i3[2])
	return val
}

type MappedMemObject struct {
	ptr        unsafe.Pointer
	size       int
	rowPitch   int
	slicePitch int
}

func (mb *MappedMemObject) ByteSlice() []byte {
	var byteSlice []byte
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&byteSlice))
	sliceHeader.Cap = mb.size
	sliceHeader.Len = mb.size
	sliceHeader.Data = uintptr(mb.ptr)
	return byteSlice
}

func (mb *MappedMemObject) Ptr() unsafe.Pointer {
	return mb.ptr
}

func (mb *MappedMemObject) Size() int {
	return mb.size
}

func (mb *MappedMemObject) RowPitch() int {
	return mb.rowPitch
}

func (mb *MappedMemObject) SlicePitch() int {
	return mb.slicePitch
}
