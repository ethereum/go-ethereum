package simulations

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

// Simulation provides a framework for running actions in a simulated network
// and then waiting for expectations to be met
type Simulation struct {
	network *Network
}

// NewSimulation returns a new simulation
func NewSimulation(network *Network) *Simulation {
	return &Simulation{
		network: network,
	}
}

// Run performs a step of the simulation by performing the step's action and
// then waiting for the step's expectation to be met
func (s *Simulation) Run(ctx context.Context, step *Step) (result *StepResult) {
	result = newStepResult()

	result.StartedAt = time.Now()
	defer func() { result.FinishedAt = time.Now() }()

	// watch network events for the duration of the step
	stop := s.watchNetwork(result)
	defer stop()

	// perform the action
	if err := step.Action(ctx); err != nil {
		result.Error = err
		return
	}

	// wait for all node expectations to either pass, error or timeout
	for len(result.Passes) < len(step.Expect.Nodes) {
		select {
		case id := <-step.Trigger:
			// skip if the node has already passed
			if _, ok := result.Passes[id]; ok {
				continue
			}

			// run the node expectation check
			pass, err := step.Expect.Check(ctx, id)
			if err != nil {
				result.Error = err
				return
			}
			if pass {
				result.Passes[id] = time.Now()
			}
		case <-ctx.Done():
			result.Error = ctx.Err()
			return
		}
	}

	return
}

func (s *Simulation) watchNetwork(result *StepResult) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	events := make(chan *Event)
	sub := s.network.Events().Subscribe(events)
	go func() {
		defer close(done)
		defer sub.Unsubscribe()
		for {
			select {
			case event := <-events:
				result.NetworkEvents = append(result.NetworkEvents, event)
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

type Step struct {
	// Action is the action to perform for this step
	Action func(context.Context) error

	// Trigger is a channel which receives node ids and triggers an
	// expectation check for that node
	Trigger chan discover.NodeID

	// Expect is the expectation to wait for when performing this step
	Expect *Expectation
}

type Expectation struct {
	// Nodes is a list of nodes to check
	Nodes []discover.NodeID

	// Check checks whether a given node meets the expectation
	Check func(context.Context, discover.NodeID) (bool, error)
}

func newStepResult() *StepResult {
	return &StepResult{
		Passes: make(map[discover.NodeID]time.Time),
	}
}

type StepResult struct {
	// Error is the error encountered whilst running the step
	Error error

	// StartedAt is the time the step started
	StartedAt time.Time

	// FinishedAt is the time the step finished
	FinishedAt time.Time

	// Passes are the timestamps of the successful node expectations
	Passes map[discover.NodeID]time.Time

	// NetworkEvents are the network events which occurred during the step
	NetworkEvents []*Event
}
