package metrics

// DiskStats is the per process disk io stats.
type DiskStats struct {
	ReadCount  int64 // Number of read operations executed
	ReadBytes  int64 // Total number of bytes read
	WriteCount int64 // Number of write operations executed
	WriteBytes int64 // Total number of byte written
}
