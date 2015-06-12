package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

// Extension: cl_APPLE_fixed_alpha_channel_orders
//
// These selectors may be passed to clCreateImage2D() in the cl_image_format.image_channel_order field.
// They are like CL_BGRA and CL_ARGB except that the alpha channel to be ignored.  On calls to read_imagef,
// the alpha will be 0xff (1.0f) if the sample falls in the image and 0 if it does not fall in the image.
// On calls to write_imagef, the alpha value is ignored and 0xff (1.0f) is written. These formats are
// currently only available for the CL_UNORM_INT8 cl_channel_type. They are intended to support legacy
// image formats.
const (
	ChannelOrder1RGBApple ChannelOrder = C.CL_1RGB_APPLE // Introduced in MacOS X.7.
	ChannelOrderBGR1Apple ChannelOrder = C.CL_BGR1_APPLE // Introduced in MacOS X.7.
)

// Extension: cl_APPLE_biased_fixed_point_image_formats
//
// This selector may be passed to clCreateImage2D() in the cl_image_format.image_channel_data_type field.
// It defines a biased signed 1.14 fixed point storage format, with range [-1, 3). The conversion from
// float to this fixed point format is defined as follows:
//
//      ushort float_to_sfixed14( float x ){
//          int i = convert_int_sat_rte( x * 0x1.0p14f );         // scale [-1, 3.0) to [-16384, 3*16384), round to nearest integer
//          i = add_sat( i, 0x4000 );                             // apply bias, to convert to [0, 65535) range
//          return convert_ushort_sat(i);                         // clamp to destination size
//      }
//
// The inverse conversion is the reverse process. The formats are currently only available on the CPU with
// the CL_RGBA channel layout.
const (
	ChannelDataTypeSFixed14Apple ChannelDataType = C.CL_SFIXED14_APPLE // Introduced in MacOS X.7.
)

func init() {
	channelOrderNameMap[ChannelOrder1RGBApple] = "1RGBApple"
	channelOrderNameMap[ChannelOrderBGR1Apple] = "RGB1Apple"
	channelDataTypeNameMap[ChannelDataTypeSFixed14Apple] = "SFixed14Apple"
}
