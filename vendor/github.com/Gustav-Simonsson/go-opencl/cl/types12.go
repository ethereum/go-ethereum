// +build cl12

package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

const (
	ChannelDataTypeUNormInt24  ChannelDataType = C.CL_UNORM_INT24
	ChannelOrderDepth          ChannelOrder    = C.CL_DEPTH
	ChannelOrderDepthStencil   ChannelOrder    = C.CL_DEPTH_STENCIL
	MemHostNoAccess            MemFlag         = C.CL_MEM_HOST_NO_ACCESS  // OpenCL 1.2
	MemHostReadOnly            MemFlag         = C.CL_MEM_HOST_READ_ONLY  // OpenCL 1.2
	MemHostWriteOnly           MemFlag         = C.CL_MEM_HOST_WRITE_ONLY // OpenCL 1.2
	MemObjectTypeImage1D       MemObjectType   = C.CL_MEM_OBJECT_IMAGE1D
	MemObjectTypeImage1DArray  MemObjectType   = C.CL_MEM_OBJECT_IMAGE1D_ARRAY
	MemObjectTypeImage1DBuffer MemObjectType   = C.CL_MEM_OBJECT_IMAGE1D_BUFFER
	MemObjectTypeImage2DArray  MemObjectType   = C.CL_MEM_OBJECT_IMAGE2D_ARRAY
	// This flag specifies that the region being mapped in the memory object is being mapped for writing.
	//
	// The contents of the region being mapped are to be discarded. This is typically the case when the
	// region being mapped is overwritten by the host. This flag allows the implementation to no longer
	// guarantee that the pointer returned by clEnqueueMapBuffer or clEnqueueMapImage contains the
	// latest bits in the region being mapped which can be a significant performance enhancement.
	MapFlagWriteInvalidateRegion MapFlag = C.CL_MAP_WRITE_INVALIDATE_REGION
)

func init() {
	errorMap[C.CL_COMPILE_PROGRAM_FAILURE] = ErrCompileProgramFailure
	errorMap[C.CL_DEVICE_PARTITION_FAILED] = ErrDevicePartitionFailed
	errorMap[C.CL_INVALID_COMPILER_OPTIONS] = ErrInvalidCompilerOptions
	errorMap[C.CL_INVALID_DEVICE_PARTITION_COUNT] = ErrInvalidDevicePartitionCount
	errorMap[C.CL_INVALID_IMAGE_DESCRIPTOR] = ErrInvalidImageDescriptor
	errorMap[C.CL_INVALID_LINKER_OPTIONS] = ErrInvalidLinkerOptions
	errorMap[C.CL_KERNEL_ARG_INFO_NOT_AVAILABLE] = ErrKernelArgInfoNotAvailable
	errorMap[C.CL_LINK_PROGRAM_FAILURE] = ErrLinkProgramFailure
	errorMap[C.CL_LINKER_NOT_AVAILABLE] = ErrLinkerNotAvailable
	channelOrderNameMap[ChannelOrderDepth] = "Depth"
	channelOrderNameMap[ChannelOrderDepthStencil] = "DepthStencil"
	channelDataTypeNameMap[ChannelDataTypeUNormInt24] = "UNormInt24"
}

type ImageDescription struct {
	Type                            MemObjectType
	Width, Height, Depth            int
	ArraySize, RowPitch, SlicePitch int
	NumMipLevels, NumSamples        int
	Buffer                          *MemObject
}

func (d ImageDescription) toCl() C.cl_image_desc {
	var desc C.cl_image_desc
	desc.image_type = C.cl_mem_object_type(d.Type)
	desc.image_width = C.size_t(d.Width)
	desc.image_height = C.size_t(d.Height)
	desc.image_depth = C.size_t(d.Depth)
	desc.image_array_size = C.size_t(d.ArraySize)
	desc.image_row_pitch = C.size_t(d.RowPitch)
	desc.image_slice_pitch = C.size_t(d.SlicePitch)
	desc.num_mip_levels = C.cl_uint(d.NumMipLevels)
	desc.num_samples = C.cl_uint(d.NumSamples)
	desc.buffer = nil
	if d.Buffer != nil {
		desc.buffer = d.Buffer.clMem
	}
	return desc
}
