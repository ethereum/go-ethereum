package testutil

import (
	"testing"

	"github.com/epiclabs-io/ut"
)

// SwarmTestTools are Swarm-specific test tools
type SwarmTestTools struct {
	*ut.TestTools
	Services *swarmTestServices
}

// BeginTest returns a project-specific test toolbox
func BeginTest(tb testing.TB, generateResults bool) *SwarmTestTools {
	ett := new(SwarmTestTools)
	ett.TestTools = ut.BeginTest(tb, generateResults)
	ett.Services = newSwarmTestServices(ett)
	return ett
}
