package cl

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

var kernelSource = `
__kernel void square(
   __global float* input,
   __global float* output,
   const unsigned int count)
{
   int i = get_global_id(0);
   if(i < count)
       output[i] = input[i] * input[i];
}
`

func getObjectStrings(object interface{}) map[string]string {
	v := reflect.ValueOf(object)
	t := reflect.TypeOf(object)

	strs := make(map[string]string)

	numMethods := t.NumMethod()
	for i := 0; i < numMethods; i++ {
		method := t.Method(i)
		if method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).Kind() == reflect.String {
			// this is a string-returning method with (presumably) only a pointer receiver parameter
			// call it
			outs := v.Method(i).Call([]reflect.Value{})
			// put the result in our map
			strs[method.Name] = (outs[0].Interface()).(string)
		}
	}

	return strs
}

func TestPlatformStringsContainNoNULs(t *testing.T) {
	platforms, err := GetPlatforms()
	if err != nil {
		t.Fatalf("Failed to get platforms: %+v", err)
	}

	for _, p := range platforms {
		for key, value := range getObjectStrings(p) {
			if strings.Contains(value, "\x00") {
				t.Fatalf("platform string %q =  %+q contains NUL", key, value)
			}
		}
	}
}

func TestDeviceStringsContainNoNULs(t *testing.T) {
	platforms, err := GetPlatforms()
	if err != nil {
		t.Fatalf("Failed to get platforms: %+v", err)
	}

	for _, p := range platforms {
		devs, err := p.GetDevices(DeviceTypeAll)
		if err != nil {
			t.Fatalf("Failed to get devices for platform %q: %+v", p.Name(), err)
		}

		for _, d := range devs {
			for key, value := range getObjectStrings(d) {
				if strings.Contains(value, "\x00") {
					t.Fatalf("device string %q =  %+q contains NUL", key, value)
				}
			}
		}
	}
}

func TestHello(t *testing.T) {
	var data [1024]float32
	for i := 0; i < len(data); i++ {
		data[i] = rand.Float32()
	}

	platforms, err := GetPlatforms()
	if err != nil {
		t.Fatalf("Failed to get platforms: %+v", err)
	}
	for i, p := range platforms {
		t.Logf("Platform %d:", i)
		t.Logf("  Name: %s", p.Name())
		t.Logf("  Vendor: %s", p.Vendor())
		t.Logf("  Profile: %s", p.Profile())
		t.Logf("  Version: %s", p.Version())
		t.Logf("  Extensions: %s", p.Extensions())
	}
	platform := platforms[0]

	devices, err := platform.GetDevices(DeviceTypeAll)
	if err != nil {
		t.Fatalf("Failed to get devices: %+v", err)
	}
	if len(devices) == 0 {
		t.Fatalf("GetDevices returned no devices")
	}
	deviceIndex := -1
	for i, d := range devices {
		if deviceIndex < 0 && d.Type() == DeviceTypeGPU {
			deviceIndex = i
		}
		t.Logf("Device %d (%s): %s", i, d.Type(), d.Name())
		t.Logf("  Address Bits: %d", d.AddressBits())
		t.Logf("  Available: %+v", d.Available())
		// t.Logf("  Built-In Kernels: %s", d.BuiltInKernels())
		t.Logf("  Compiler Available: %+v", d.CompilerAvailable())
		t.Logf("  Double FP Config: %s", d.DoubleFPConfig())
		t.Logf("  Driver Version: %s", d.DriverVersion())
		t.Logf("  Error Correction Supported: %+v", d.ErrorCorrectionSupport())
		t.Logf("  Execution Capabilities: %s", d.ExecutionCapabilities())
		t.Logf("  Extensions: %s", d.Extensions())
		t.Logf("  Global Memory Cache Type: %s", d.GlobalMemCacheType())
		t.Logf("  Global Memory Cacheline Size: %d KB", d.GlobalMemCachelineSize()/1024)
		t.Logf("  Global Memory Size: %d MB", d.GlobalMemSize()/(1024*1024))
		t.Logf("  Half FP Config: %s", d.HalfFPConfig())
		t.Logf("  Host Unified Memory: %+v", d.HostUnifiedMemory())
		t.Logf("  Image Support: %+v", d.ImageSupport())
		t.Logf("  Image2D Max Dimensions: %d x %d", d.Image2DMaxWidth(), d.Image2DMaxHeight())
		t.Logf("  Image3D Max Dimenionns: %d x %d x %d", d.Image3DMaxWidth(), d.Image3DMaxHeight(), d.Image3DMaxDepth())
		// t.Logf("  Image Max Buffer Size: %d", d.ImageMaxBufferSize())
		// t.Logf("  Image Max Array Size: %d", d.ImageMaxArraySize())
		// t.Logf("  Linker Available: %+v", d.LinkerAvailable())
		t.Logf("  Little Endian: %+v", d.EndianLittle())
		t.Logf("  Local Mem Size Size: %d KB", d.LocalMemSize()/1024)
		t.Logf("  Local Mem Type: %s", d.LocalMemType())
		t.Logf("  Max Clock Frequency: %d", d.MaxClockFrequency())
		t.Logf("  Max Compute Units: %d", d.MaxComputeUnits())
		t.Logf("  Max Constant Args: %d", d.MaxConstantArgs())
		t.Logf("  Max Constant Buffer Size: %d KB", d.MaxConstantBufferSize()/1024)
		t.Logf("  Max Mem Alloc Size: %d KB", d.MaxMemAllocSize()/1024)
		t.Logf("  Max Parameter Size: %d", d.MaxParameterSize())
		t.Logf("  Max Read-Image Args: %d", d.MaxReadImageArgs())
		t.Logf("  Max Samplers: %d", d.MaxSamplers())
		t.Logf("  Max Work Group Size: %d", d.MaxWorkGroupSize())
		t.Logf("  Max Work Item Dimensions: %d", d.MaxWorkItemDimensions())
		t.Logf("  Max Work Item Sizes: %d", d.MaxWorkItemSizes())
		t.Logf("  Max Write-Image Args: %d", d.MaxWriteImageArgs())
		t.Logf("  Memory Base Address Alignment: %d", d.MemBaseAddrAlign())
		t.Logf("  Native Vector Width Char: %d", d.NativeVectorWidthChar())
		t.Logf("  Native Vector Width Short: %d", d.NativeVectorWidthShort())
		t.Logf("  Native Vector Width Int: %d", d.NativeVectorWidthInt())
		t.Logf("  Native Vector Width Long: %d", d.NativeVectorWidthLong())
		t.Logf("  Native Vector Width Float: %d", d.NativeVectorWidthFloat())
		t.Logf("  Native Vector Width Double: %d", d.NativeVectorWidthDouble())
		t.Logf("  Native Vector Width Half: %d", d.NativeVectorWidthHalf())
		t.Logf("  OpenCL C Version: %s", d.OpenCLCVersion())
		// t.Logf("  Parent Device: %+v", d.ParentDevice())
		t.Logf("  Profile: %s", d.Profile())
		t.Logf("  Profiling Timer Resolution: %d", d.ProfilingTimerResolution())
		t.Logf("  Vendor: %s", d.Vendor())
		t.Logf("  Version: %s", d.Version())
	}
	if deviceIndex < 0 {
		deviceIndex = 0
	}
	device := devices[deviceIndex]
	t.Logf("Using device %d", deviceIndex)
	context, err := CreateContext([]*Device{device})
	if err != nil {
		t.Fatalf("CreateContext failed: %+v", err)
	}
	// imageFormats, err := context.GetSupportedImageFormats(0, MemObjectTypeImage2D)
	// if err != nil {
	// 	t.Fatalf("GetSupportedImageFormats failed: %+v", err)
	// }
	// t.Logf("Supported image formats: %+v", imageFormats)
	queue, err := context.CreateCommandQueue(device, 0)
	if err != nil {
		t.Fatalf("CreateCommandQueue failed: %+v", err)
	}
	program, err := context.CreateProgramWithSource([]string{kernelSource})
	if err != nil {
		t.Fatalf("CreateProgramWithSource failed: %+v", err)
	}
	if err := program.BuildProgram(nil, ""); err != nil {
		t.Fatalf("BuildProgram failed: %+v", err)
	}
	kernel, err := program.CreateKernel("square")
	if err != nil {
		t.Fatalf("CreateKernel failed: %+v", err)
	}
	for i := 0; i < 3; i++ {
		name, err := kernel.ArgName(i)
		if err == ErrUnsupported {
			break
		} else if err != nil {
			t.Errorf("GetKernelArgInfo for name failed: %+v", err)
			break
		} else {
			t.Logf("Kernel arg %d: %s", i, name)
		}
	}
	input, err := context.CreateEmptyBuffer(MemReadOnly, 4*len(data))
	if err != nil {
		t.Fatalf("CreateBuffer failed for input: %+v", err)
	}
	output, err := context.CreateEmptyBuffer(MemReadOnly, 4*len(data))
	if err != nil {
		t.Fatalf("CreateBuffer failed for output: %+v", err)
	}
	if _, err := queue.EnqueueWriteBufferFloat32(input, true, 0, data[:], nil); err != nil {
		t.Fatalf("EnqueueWriteBufferFloat32 failed: %+v", err)
	}
	if err := kernel.SetArgs(input, output, uint32(len(data))); err != nil {
		t.Fatalf("SetKernelArgs failed: %+v", err)
	}

	local, err := kernel.WorkGroupSize(device)
	if err != nil {
		t.Fatalf("WorkGroupSize failed: %+v", err)
	}
	t.Logf("Work group size: %d", local)
	size, _ := kernel.PreferredWorkGroupSizeMultiple(nil)
	t.Logf("Preferred Work Group Size Multiple: %d", size)

	global := len(data)
	d := len(data) % local
	if d != 0 {
		global += local - d
	}
	if _, err := queue.EnqueueNDRangeKernel(kernel, nil, []int{global}, []int{local}, nil); err != nil {
		t.Fatalf("EnqueueNDRangeKernel failed: %+v", err)
	}

	if err := queue.Finish(); err != nil {
		t.Fatalf("Finish failed: %+v", err)
	}

	results := make([]float32, len(data))
	if _, err := queue.EnqueueReadBufferFloat32(output, true, 0, results, nil); err != nil {
		t.Fatalf("EnqueueReadBufferFloat32 failed: %+v", err)
	}

	correct := 0
	for i, v := range data {
		if results[i] == v*v {
			correct++
		}
	}

	if correct != len(data) {
		t.Fatalf("%d/%d correct values", correct, len(data))
	}
}
