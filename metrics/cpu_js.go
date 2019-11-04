// +build js,wasm

package metrics

// getProcessCPUTime is mocked for js/wasm environments. Currently it alwasy
// returns 0.
func getProcessCPUTime() int64 {
	return 0
}
