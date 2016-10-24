package cl

// #ifdef __APPLE__
// #include "OpenCL/opencl.h"
// #else
// #include "cl.h"
// #endif
import "C"

import "unsafe"

const maxPlatforms = 32

type Platform struct {
	id C.cl_platform_id
}

// Obtain the list of platforms available.
func GetPlatforms() ([]*Platform, error) {
	var platformIds [maxPlatforms]C.cl_platform_id
	var nPlatforms C.cl_uint
	if err := C.clGetPlatformIDs(C.cl_uint(maxPlatforms), &platformIds[0], &nPlatforms); err != C.CL_SUCCESS {
		return nil, toError(err)
	}
	platforms := make([]*Platform, nPlatforms)
	for i := 0; i < int(nPlatforms); i++ {
		platforms[i] = &Platform{id: platformIds[i]}
	}
	return platforms, nil
}

func (p *Platform) GetDevices(deviceType DeviceType) ([]*Device, error) {
	return GetDevices(p, deviceType)
}

func (p *Platform) getInfoString(param C.cl_platform_info) (string, error) {
	var strC [2048]byte
	var strN C.size_t
	if err := C.clGetPlatformInfo(p.id, param, 2048, unsafe.Pointer(&strC[0]), &strN); err != C.CL_SUCCESS {
		return "", toError(err)
	}
	return string(strC[:(strN - 1)]), nil
}

func (p *Platform) Name() string {
	if str, err := p.getInfoString(C.CL_PLATFORM_NAME); err != nil {
		panic("Platform.Name() should never fail")
	} else {
		return str
	}
}

func (p *Platform) Vendor() string {
	if str, err := p.getInfoString(C.CL_PLATFORM_VENDOR); err != nil {
		panic("Platform.Vendor() should never fail")
	} else {
		return str
	}
}

func (p *Platform) Profile() string {
	if str, err := p.getInfoString(C.CL_PLATFORM_PROFILE); err != nil {
		panic("Platform.Profile() should never fail")
	} else {
		return str
	}
}

func (p *Platform) Version() string {
	if str, err := p.getInfoString(C.CL_PLATFORM_VERSION); err != nil {
		panic("Platform.Version() should never fail")
	} else {
		return str
	}
}

func (p *Platform) Extensions() string {
	if str, err := p.getInfoString(C.CL_PLATFORM_EXTENSIONS); err != nil {
		panic("Platform.Extensions() should never fail")
	} else {
		return str
	}
}
