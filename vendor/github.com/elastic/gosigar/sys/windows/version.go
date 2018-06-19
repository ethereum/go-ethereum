// +build windows

package windows

import (
	"fmt"
	"syscall"
)

// Version identifies a Windows version by major, minor, and build number.
type Version struct {
	Major int
	Minor int
	Build int
}

// GetWindowsVersion returns the Windows version information. Applications not
// manifested for Windows 8.1 or Windows 10 will return the Windows 8 OS version
// value (6.2).
//
// For a table of version numbers see:
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724833(v=vs.85).aspx
func GetWindowsVersion() Version {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724439(v=vs.85).aspx
	ver, err := syscall.GetVersion()
	if err != nil {
		// GetVersion should never return an error.
		panic(fmt.Errorf("GetVersion failed: %v", err))
	}

	return Version{
		Major: int(ver & 0xFF),
		Minor: int(ver >> 8 & 0xFF),
		Build: int(ver >> 16),
	}
}

// IsWindowsVistaOrGreater returns true if the Windows version is Vista or
// greater.
func (v Version) IsWindowsVistaOrGreater() bool {
	// Vista is 6.0.
	return v.Major >= 6 && v.Minor >= 0
}
