package testutil

import (
	"github.com/epiclabs-io/ut"
)

// SwarmTestServices groups all third-party dependencies to provide easy instantiation
// and cleanup of these. By default, it provides temporary files and directories
type swarmTestServices struct {
	tt *SwarmTestTools
	*ut.FileServices
}

// Instantiate Test services
func newSwarmTestServices(tt *SwarmTestTools) *swarmTestServices {
	return &swarmTestServices{
		tt:           tt,
		FileServices: ut.NewFileServices(tt.TestTools),
	}
}
