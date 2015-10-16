// +build cl12

package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"
import (
	"image"
	"unsafe"
)

func (ctx *Context) CreateImage(flags MemFlag, imageFormat ImageFormat, imageDesc ImageDescription, data []byte) (*MemObject, error) {
	format := imageFormat.toCl()
	desc := imageDesc.toCl()
	var dataPtr unsafe.Pointer
	if data != nil {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var err C.cl_int
	clBuffer := C.clCreateImage(ctx.clContext, C.cl_mem_flags(flags), &format, &desc, dataPtr, &err)
	if err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	if clBuffer == nil {
		return nil, ErrUnknown
	}
	return newMemObject(clBuffer, len(data)), nil
}

func (ctx *Context) CreateImageSimple(flags MemFlag, width, height int, channelOrder ChannelOrder, channelDataType ChannelDataType, data []byte) (*MemObject, error) {
	format := ImageFormat{channelOrder, channelDataType}
	desc := ImageDescription{
		Type:   MemObjectTypeImage2D,
		Width:  width,
		Height: height,
	}
	return ctx.CreateImage(flags, format, desc, data)
}

func (ctx *Context) CreateImageFromImage(flags MemFlag, img image.Image) (*MemObject, error) {
	switch m := img.(type) {
	case *image.Gray:
		format := ImageFormat{ChannelOrderIntensity, ChannelDataTypeUNormInt8}
		desc := ImageDescription{
			Type:     MemObjectTypeImage2D,
			Width:    m.Bounds().Dx(),
			Height:   m.Bounds().Dy(),
			RowPitch: m.Stride,
		}
		return ctx.CreateImage(flags, format, desc, m.Pix)
	case *image.RGBA:
		format := ImageFormat{ChannelOrderRGBA, ChannelDataTypeUNormInt8}
		desc := ImageDescription{
			Type:     MemObjectTypeImage2D,
			Width:    m.Bounds().Dx(),
			Height:   m.Bounds().Dy(),
			RowPitch: m.Stride,
		}
		return ctx.CreateImage(flags, format, desc, m.Pix)
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	data := make([]byte, w*h*4)
	dataOffset := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x+b.Min.X, y+b.Min.Y)
			r, g, b, a := c.RGBA()
			data[dataOffset] = uint8(r >> 8)
			data[dataOffset+1] = uint8(g >> 8)
			data[dataOffset+2] = uint8(b >> 8)
			data[dataOffset+3] = uint8(a >> 8)
			dataOffset += 4
		}
	}
	return ctx.CreateImageSimple(flags, w, h, ChannelOrderRGBA, ChannelDataTypeUNormInt8, data)
}
