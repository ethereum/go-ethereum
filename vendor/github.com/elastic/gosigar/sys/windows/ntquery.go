// +build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// On both 32-bit and 64-bit systems NtQuerySystemInformation expects the
// size of SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION to be 48.
const sizeofSystemProcessorPerformanceInformation = 48

// ProcessBasicInformation is an equivalent representation of
// PROCESS_BASIC_INFORMATION in the Windows API.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684280(v=vs.85).aspx
type ProcessBasicInformation struct {
	ExitStatus                   uint
	PebBaseAddress               uintptr
	AffinityMask                 uint
	BasePriority                 uint
	UniqueProcessID              uint
	InheritedFromUniqueProcessID uint
}

// NtQueryProcessBasicInformation queries basic information about the process
// associated with the given handle (provided by OpenProcess). It uses the
// NtQueryInformationProcess function to collect the data.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684280(v=vs.85).aspx
func NtQueryProcessBasicInformation(handle syscall.Handle) (ProcessBasicInformation, error) {
	var processBasicInfo ProcessBasicInformation
	processBasicInfoPtr := (*byte)(unsafe.Pointer(&processBasicInfo))
	size := uint32(unsafe.Sizeof(processBasicInfo))
	ntStatus, _ := _NtQueryInformationProcess(handle, 0, processBasicInfoPtr, size, nil)
	if ntStatus != 0 {
		return ProcessBasicInformation{}, errors.Errorf("NtQueryInformationProcess failed, NTSTATUS=0x%X", ntStatus)
	}

	return processBasicInfo, nil
}

// SystemProcessorPerformanceInformation contains CPU performance information
// for a single CPU.
type SystemProcessorPerformanceInformation struct {
	IdleTime   time.Duration // Amount of time spent idle.
	KernelTime time.Duration // Kernel time does NOT include time spent in idle.
	UserTime   time.Duration // Amount of time spent executing in user mode.
}

// _SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION is an equivalent representation of
// SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION in the Windows API. This struct is
// used internally with NtQuerySystemInformation call and is not exported. The
// exported equivalent is SystemProcessorPerformanceInformation.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724509(v=vs.85).aspx
type _SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION struct {
	IdleTime   int64
	KernelTime int64
	UserTime   int64
	Reserved1  [2]int64
	Reserved2  uint32
}

// NtQuerySystemProcessorPerformanceInformation queries CPU performance
// information for each CPU. It uses the NtQuerySystemInformation function to
// collect the SystemProcessorPerformanceInformation.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724509(v=vs.85).aspx
func NtQuerySystemProcessorPerformanceInformation() ([]SystemProcessorPerformanceInformation, error) {
	// NTSTATUS code for success.
	// https://msdn.microsoft.com/en-us/library/cc704588.aspx
	const STATUS_SUCCESS = 0

	// From the _SYSTEM_INFORMATION_CLASS enum.
	// http://processhacker.sourceforge.net/doc/ntexapi_8h.html#ad5d815b48e8f4da1ef2eb7a2f18a54e0
	const systemProcessorPerformanceInformation = 8

	// Create a buffer large enough to hold an entry for each processor.
	b := make([]byte, runtime.NumCPU()*sizeofSystemProcessorPerformanceInformation)

	// Query the performance information. Note that this function uses 0 to
	// indicate success. Most other Windows functions use non-zero for success.
	var returnLength uint32
	ntStatus, _ := _NtQuerySystemInformation(systemProcessorPerformanceInformation, &b[0], uint32(len(b)), &returnLength)
	if ntStatus != STATUS_SUCCESS {
		return nil, errors.Errorf("NtQuerySystemInformation failed, NTSTATUS=0x%X, bufLength=%v, returnLength=%v", ntStatus, len(b), returnLength)
	}

	return readSystemProcessorPerformanceInformationBuffer(b)
}

// readSystemProcessorPerformanceInformationBuffer reads from a buffer
// containing SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION data. The buffer should
// contain one entry for each CPU.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724509(v=vs.85).aspx
func readSystemProcessorPerformanceInformationBuffer(b []byte) ([]SystemProcessorPerformanceInformation, error) {
	n := len(b) / sizeofSystemProcessorPerformanceInformation
	r := bytes.NewReader(b)

	rtn := make([]SystemProcessorPerformanceInformation, 0, n)
	for i := 0; i < n; i++ {
		_, err := r.Seek(int64(i*sizeofSystemProcessorPerformanceInformation), io.SeekStart)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to seek to cpuN=%v in buffer", i)
		}

		times := make([]uint64, 3)
		for j := range times {
			err := binary.Read(r, binary.LittleEndian, &times[j])
			if err != nil {
				return nil, errors.Wrapf(err, "failed reading cpu times for cpuN=%v", i)
			}
		}

		idleTime := time.Duration(times[0] * 100)
		kernelTime := time.Duration(times[1] * 100)
		userTime := time.Duration(times[2] * 100)

		rtn = append(rtn, SystemProcessorPerformanceInformation{
			IdleTime:   idleTime,
			KernelTime: kernelTime - idleTime, // Subtract out idle time from kernel time.
			UserTime:   userTime,
		})
	}

	return rtn, nil
}
