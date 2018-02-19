package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	sizeofUint32                  = 4
	sizeofProcessEntry32          = uint32(unsafe.Sizeof(ProcessEntry32{}))
	sizeofProcessMemoryCountersEx = uint32(unsafe.Sizeof(ProcessMemoryCountersEx{}))
	sizeofMemoryStatusEx          = uint32(unsafe.Sizeof(MemoryStatusEx{}))
)

// Process-specific access rights. Others are declared in the syscall package.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684880(v=vs.85).aspx
const (
	PROCESS_QUERY_LIMITED_INFORMATION uint32 = 0x1000
	PROCESS_VM_READ                   uint32 = 0x0010
)

// MAX_PATH is the maximum length for a path in Windows.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
const MAX_PATH = 260

// DriveType represents a type of drive (removable, fixed, CD-ROM, RAM disk, or
// network drive).
type DriveType uint32

// Drive types as returned by GetDriveType.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa364939(v=vs.85).aspx
const (
	DRIVE_UNKNOWN DriveType = iota
	DRIVE_NO_ROOT_DIR
	DRIVE_REMOVABLE
	DRIVE_FIXED
	DRIVE_REMOTE
	DRIVE_CDROM
	DRIVE_RAMDISK
)

func (dt DriveType) String() string {
	names := map[DriveType]string{
		DRIVE_UNKNOWN:     "unknown",
		DRIVE_NO_ROOT_DIR: "invalid",
		DRIVE_REMOVABLE:   "removable",
		DRIVE_FIXED:       "fixed",
		DRIVE_REMOTE:      "remote",
		DRIVE_CDROM:       "cdrom",
		DRIVE_RAMDISK:     "ramdisk",
	}

	name, found := names[dt]
	if !found {
		return "unknown DriveType value"
	}
	return name
}

// Flags that can be used with CreateToolhelp32Snapshot.
const (
	TH32CS_INHERIT      uint32 = 0x80000000 // Indicates that the snapshot handle is to be inheritable.
	TH32CS_SNAPHEAPLIST uint32 = 0x00000001 // Includes all heaps of the process specified in th32ProcessID in the snapshot.
	TH32CS_SNAPMODULE   uint32 = 0x00000008 // Includes all modules of the process specified in th32ProcessID in the snapshot.
	TH32CS_SNAPMODULE32 uint32 = 0x00000010 // Includes all 32-bit modules of the process specified in th32ProcessID in the snapshot when called from a 64-bit process.
	TH32CS_SNAPPROCESS  uint32 = 0x00000002 // Includes all processes in the system in the snapshot.
	TH32CS_SNAPTHREAD   uint32 = 0x00000004 // Includes all threads in the system in the snapshot.
)

// ProcessEntry32 is an equivalent representation of PROCESSENTRY32 in the
// Windows API. It contains a process's information. Do not modify or reorder.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684839(v=vs.85).aspx
type ProcessEntry32 struct {
	size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	exeFile           [MAX_PATH]uint16
}

// ExeFile returns the name of the executable file for the process. It does
// not contain the full path.
func (p ProcessEntry32) ExeFile() string {
	return syscall.UTF16ToString(p.exeFile[:])
}

func (p ProcessEntry32) String() string {
	return fmt.Sprintf("{CntUsage:%v ProcessID:%v DefaultHeapID:%v ModuleID:%v "+
		"CntThreads:%v ParentProcessID:%v PriorityClassBase:%v Flags:%v ExeFile:%v",
		p.CntUsage, p.ProcessID, p.DefaultHeapID, p.ModuleID, p.CntThreads,
		p.ParentProcessID, p.PriorityClassBase, p.Flags, p.ExeFile())
}

// MemoryStatusEx is an equivalent representation of MEMORYSTATUSEX in the
// Windows API. It contains information about the current state of both physical
// and virtual memory, including extended memory.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366770
type MemoryStatusEx struct {
	length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// ProcessMemoryCountersEx is an equivalent representation of
// PROCESS_MEMORY_COUNTERS_EX in the Windows API.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684874(v=vs.85).aspx
type ProcessMemoryCountersEx struct {
	cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

// GetLogicalDriveStrings returns a list of drives in the system.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa364975(v=vs.85).aspx
func GetLogicalDriveStrings() ([]string, error) {
	// Determine the size of the buffer required to receive all drives.
	bufferLength, err := _GetLogicalDriveStringsW(0, nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetLogicalDriveStringsW failed to get buffer length")
	}
	if bufferLength < 0 {
		return nil, errors.New("GetLogicalDriveStringsW returned an invalid buffer length")
	}

	buffer := make([]uint16, bufferLength)
	_, err = _GetLogicalDriveStringsW(uint32(len(buffer)), &buffer[0])
	if err != nil {
		return nil, errors.Wrap(err, "GetLogicalDriveStringsW failed")
	}

	// Split the uint16 slice at null-terminators.
	var startIdx int
	var drivesUTF16 [][]uint16
	for i, value := range buffer {
		if value == 0 {
			drivesUTF16 = append(drivesUTF16, buffer[startIdx:i])
			startIdx = i + 1
		}
	}

	// Convert the utf16 slices to strings.
	drives := make([]string, 0, len(drivesUTF16))
	for _, driveUTF16 := range drivesUTF16 {
		if len(driveUTF16) > 0 {
			drives = append(drives, syscall.UTF16ToString(driveUTF16))
		}
	}

	return drives, nil
}

// GlobalMemoryStatusEx retrieves information about the system's current usage
// of both physical and virtual memory.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366589(v=vs.85).aspx
func GlobalMemoryStatusEx() (MemoryStatusEx, error) {
	memoryStatusEx := MemoryStatusEx{length: sizeofMemoryStatusEx}
	err := _GlobalMemoryStatusEx(&memoryStatusEx)
	if err != nil {
		return MemoryStatusEx{}, errors.Wrap(err, "GlobalMemoryStatusEx failed")
	}

	return memoryStatusEx, nil
}

// GetProcessMemoryInfo retrieves information about the memory usage of the
// specified process.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683219(v=vs.85).aspx
func GetProcessMemoryInfo(handle syscall.Handle) (ProcessMemoryCountersEx, error) {
	processMemoryCountersEx := ProcessMemoryCountersEx{cb: sizeofProcessMemoryCountersEx}
	err := _GetProcessMemoryInfo(handle, &processMemoryCountersEx, processMemoryCountersEx.cb)
	if err != nil {
		return ProcessMemoryCountersEx{}, errors.Wrap(err, "GetProcessMemoryInfo failed")
	}

	return processMemoryCountersEx, nil
}

// GetProcessImageFileName Retrieves the name of the executable file for the
// specified process.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683217(v=vs.85).aspx
func GetProcessImageFileName(handle syscall.Handle) (string, error) {
	buffer := make([]uint16, MAX_PATH)
	_, err := _GetProcessImageFileName(handle, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", errors.Wrap(err, "GetProcessImageFileName failed")
	}

	return syscall.UTF16ToString(buffer), nil
}

// GetSystemTimes retrieves system timing information. On a multiprocessor
// system, the values returned are the sum of the designated times across all
// processors. The returned kernel time does not include the system idle time.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724400(v=vs.85).aspx
func GetSystemTimes() (idle, kernel, user time.Duration, err error) {
	var idleTime, kernelTime, userTime syscall.Filetime
	err = _GetSystemTimes(&idleTime, &kernelTime, &userTime)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "GetSystemTimes failed")
	}

	idle = FiletimeToDuration(&idleTime)
	kernel = FiletimeToDuration(&kernelTime) // Kernel time includes idle time so we subtract it out.
	user = FiletimeToDuration(&userTime)

	return idle, kernel - idle, user, nil
}

// FiletimeToDuration converts a Filetime to a time.Duration. Do not use this
// method to convert a Filetime to an actual clock time, for that use
// Filetime.Nanosecond().
func FiletimeToDuration(ft *syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime) // in 100-nanosecond intervals
	return time.Duration(n * 100)
}

// GetDriveType Determines whether a disk drive is a removable, fixed, CD-ROM,
// RAM disk, or network drive. A trailing backslash is required on the
// rootPathName.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa364939
func GetDriveType(rootPathName string) (DriveType, error) {
	rootPathNamePtr, err := syscall.UTF16PtrFromString(rootPathName)
	if err != nil {
		return DRIVE_UNKNOWN, errors.Wrapf(err, "UTF16PtrFromString failed for rootPathName=%v", rootPathName)
	}

	dt, err := _GetDriveType(rootPathNamePtr)
	if err != nil {
		return DRIVE_UNKNOWN, errors.Wrapf(err, "GetDriveType failed for rootPathName=%v", rootPathName)
	}

	return dt, nil
}

// EnumProcesses retrieves the process identifier for each process object in the
// system. This function can return a max of 65536 PIDs. If there are more
// processes than that then this will not return them all.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms682629(v=vs.85).aspx
func EnumProcesses() ([]uint32, error) {
	enumProcesses := func(size int) ([]uint32, error) {
		var (
			pids         = make([]uint32, size)
			sizeBytes    = len(pids) * sizeofUint32
			bytesWritten uint32
		)

		err := _EnumProcesses(&pids[0], uint32(sizeBytes), &bytesWritten)

		pidsWritten := int(bytesWritten) / sizeofUint32
		if int(bytesWritten)%sizeofUint32 != 0 || pidsWritten > len(pids) {
			return nil, errors.Errorf("EnumProcesses returned an invalid bytesWritten value of %v", bytesWritten)
		}
		pids = pids[:pidsWritten]

		return pids, err
	}

	// Retry the EnumProcesses call with larger arrays if needed.
	size := 2048
	var pids []uint32
	for tries := 0; tries < 5; tries++ {
		var err error
		pids, err = enumProcesses(size)
		if err != nil {
			return nil, errors.Wrap(err, "EnumProcesses failed")
		}

		if len(pids) < size {
			break
		}

		// Increase the size the pids array and retry the enumProcesses call
		// because the array wasn't large enough to hold all of the processes.
		size *= 2
	}

	return pids, nil
}

// GetDiskFreeSpaceEx retrieves information about the amount of space that is
// available on a disk volume, which is the total amount of space, the total
// amount of free space, and the total amount of free space available to the
// user that is associated with the calling thread.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa364937(v=vs.85).aspx
func GetDiskFreeSpaceEx(directoryName string) (freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64, err error) {
	directoryNamePtr, err := syscall.UTF16PtrFromString(directoryName)
	if err != nil {
		return 0, 0, 0, errors.Wrapf(err, "UTF16PtrFromString failed for directoryName=%v", directoryName)
	}

	err = _GetDiskFreeSpaceEx(directoryNamePtr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, 0, 0, err
	}

	return freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes, nil
}

// CreateToolhelp32Snapshot takes a snapshot of the specified processes, as well
// as the heaps, modules, and threads used by these processes.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms682489(v=vs.85).aspx
func CreateToolhelp32Snapshot(flags, pid uint32) (syscall.Handle, error) {
	h, err := _CreateToolhelp32Snapshot(flags, pid)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	if h == syscall.InvalidHandle {
		return syscall.InvalidHandle, syscall.GetLastError()
	}

	return h, nil
}

// Process32First retrieves information about the first process encountered in a
// system snapshot.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684834
func Process32First(handle syscall.Handle) (ProcessEntry32, error) {
	processEntry32 := ProcessEntry32{size: sizeofProcessEntry32}
	err := _Process32First(handle, &processEntry32)
	if err != nil {
		return ProcessEntry32{}, errors.Wrap(err, "Process32First failed")
	}

	return processEntry32, nil
}

// Process32Next retrieves information about the next process recorded in a
// system snapshot. When there are no more processes to iterate then
// syscall.ERROR_NO_MORE_FILES is returned (use errors.Cause() to unwrap).
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684836
func Process32Next(handle syscall.Handle) (ProcessEntry32, error) {
	processEntry32 := ProcessEntry32{size: sizeofProcessEntry32}
	err := _Process32Next(handle, &processEntry32)
	if err != nil {
		return ProcessEntry32{}, errors.Wrap(err, "Process32Next failed")
	}

	return processEntry32, nil
}

// Use "GOOS=windows go generate -v -x ." to generate the source.

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   _GlobalMemoryStatusEx(buffer *MemoryStatusEx) (err error) = kernel32.GlobalMemoryStatusEx
//sys   _GetLogicalDriveStringsW(bufferLength uint32, buffer *uint16) (length uint32, err error) = kernel32.GetLogicalDriveStringsW
//sys   _GetProcessMemoryInfo(handle syscall.Handle, psmemCounters *ProcessMemoryCountersEx, cb uint32) (err error) = psapi.GetProcessMemoryInfo
//sys   _GetProcessImageFileName(handle syscall.Handle, outImageFileName *uint16, size uint32) (length uint32, err error) = psapi.GetProcessImageFileNameW
//sys   _GetSystemTimes(idleTime *syscall.Filetime, kernelTime *syscall.Filetime, userTime *syscall.Filetime) (err error) = kernel32.GetSystemTimes
//sys   _GetDriveType(rootPathName *uint16) (dt DriveType, err error) = kernel32.GetDriveTypeW
//sys   _EnumProcesses(processIds *uint32, sizeBytes uint32, bytesReturned *uint32) (err error) = psapi.EnumProcesses
//sys   _GetDiskFreeSpaceEx(directoryName *uint16, freeBytesAvailable *uint64, totalNumberOfBytes *uint64, totalNumberOfFreeBytes *uint64) (err error) = kernel32.GetDiskFreeSpaceExW
//sys   _Process32First(handle syscall.Handle, processEntry32 *ProcessEntry32) (err error) = kernel32.Process32FirstW
//sys   _Process32Next(handle syscall.Handle, processEntry32 *ProcessEntry32) (err error) = kernel32.Process32NextW
//sys   _CreateToolhelp32Snapshot(flags uint32, processID uint32) (handle syscall.Handle, err error) = kernel32.CreateToolhelp32Snapshot
//sys   _NtQuerySystemInformation(systemInformationClass uint32, systemInformation *byte, systemInformationLength uint32, returnLength *uint32) (ntstatus uint32, err error) = ntdll.NtQuerySystemInformation
//sys   _NtQueryInformationProcess(processHandle syscall.Handle, processInformationClass uint32, processInformation *byte, processInformationLength uint32, returnLength *uint32) (ntstatus uint32, err error) = ntdll.NtQueryInformationProcess
//sys   _LookupPrivilegeName(systemName string, luid *int64, buffer *uint16, size *uint32) (err error) = advapi32.LookupPrivilegeNameW
//sys   _LookupPrivilegeValue(systemName string, name string, luid *int64) (err error) = advapi32.LookupPrivilegeValueW
//sys   _AdjustTokenPrivileges(token syscall.Token, releaseAll bool, input *byte, outputSize uint32, output *byte, requiredSize *uint32) (success bool, err error) [true] = advapi32.AdjustTokenPrivileges
