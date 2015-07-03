// +build !linux

package metrics

import "errors"

// ReadDiskStats retrieves the disk IO stats belonging to the current process.
func ReadDiskStats(stats *DiskStats) error {
	return errors.New("Not implemented")
}
