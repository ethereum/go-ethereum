// Contains the Linux implementation of process disk IO counter retrieval.

package metrics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ReadDiskStats retrieves the disk IO stats belonging to the current process.
func ReadDiskStats(stats *DiskStats) error {
	// Open the process disk IO counter file
	inf, err := os.Open(fmt.Sprintf("/proc/%d/io", os.Getpid()))
	if err != nil {
		return err
	}
	in := bufio.NewReader(inf)

	// Iterate over the IO counter, and extract what we need
	for {
		// Read the next line and split to key and value
		line, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		key, value := "", int64(0)
		if parts := strings.Split(line, ":"); len(parts) != 2 {
			continue
		} else {
			key = strings.TrimSpace(parts[0])
			if value, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err != nil {
				return err
			}
		}
		// Update the counter based on the key
		switch key {
		case "syscr":
			stats.ReadCount = value
		case "syscw":
			stats.WriteCount = value
		case "rchar":
			stats.ReadBytes = value
		case "wchar":
			stats.WriteBytes = value
		}
	}
	return nil
}
