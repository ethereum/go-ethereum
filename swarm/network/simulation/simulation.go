// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package simulation

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// Common errors that are returned by functions in this package.
var (
	ErrNodeNotFound = errors.New("node not found")
)

// Simulation provides methods on network, nodes and services
// to manage them.
type Simulation struct {
	// Net is exposed as a way to access lower level functionalities
	// of p2p/simulations.Network.
	Net *simulations.Network

	serviceNames      []string
	cleanupFuncs      []func()
	buckets           map[enode.ID]*sync.Map
	shutdownWG        sync.WaitGroup
	done              chan struct{}
	mu                sync.RWMutex
	neighbourhoodSize int

	httpSrv *http.Server        //attach a HTTP server via SimulationOptions
	handler *simulations.Server //HTTP handler for the server
	runC    chan struct{}       //channel where frontend signals it is ready
}

// ServiceFunc is used in New to declare new service constructor.
// The first argument provides ServiceContext from the adapters package
// giving for example the access to NodeID. Second argument is the sync.Map
// where all "global" state related to the service should be kept.
// All cleanups needed for constructed service and any other constructed
// objects should ne provided in a single returned cleanup function.
// Returned cleanup function will be called by Close function
// after network shutdown.
type ServiceFunc func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error)

// New creates a new simulation instance
// Services map must have unique keys as service names and
// every ServiceFunc must return a node.Service of the unique type.
// This restriction is required by node.Node.Start() function
// which is used to start node.Service returned by ServiceFunc.
func New(services map[string]ServiceFunc) (s *Simulation) {
	s = &Simulation{
		buckets:           make(map[enode.ID]*sync.Map),
		done:              make(chan struct{}),
		neighbourhoodSize: network.NewKadParams().NeighbourhoodSize,
	}

	adapterServices := make(map[string]adapters.ServiceFunc, len(services))
	for name, serviceFunc := range services {
		// Scope this variables correctly
		// as they will be in the adapterServices[name] function accessed later.
		name, serviceFunc := name, serviceFunc
		s.serviceNames = append(s.serviceNames, name)
		adapterServices[name] = func(ctx *adapters.ServiceContext) (node.Service, error) {
			b := new(sync.Map)
			service, cleanup, err := serviceFunc(ctx, b)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if cleanup != nil {
				s.cleanupFuncs = append(s.cleanupFuncs, cleanup)
			}
			s.buckets[ctx.Config.ID] = b
			return service, nil
		}
	}

	s.Net = simulations.NewNetwork(
		adapters.NewTCPAdapter(adapterServices),
		&simulations.NetworkConfig{ID: "0"},
	)

	return s
}

// RunFunc is the function that will be called
// on Simulation.Run method call.
type RunFunc func(context.Context, *Simulation) error

// Result is the returned value of Simulation.Run method.
type Result struct {
	Duration time.Duration
	Error    error
}

// Run calls the RunFunc function while taking care of
// cancellation provided through the Context.
func (s *Simulation) Run(ctx context.Context, f RunFunc) (r Result) {
	//if the option is set to run a HTTP server with the simulation,
	//init the server and start it
	start := time.Now()
	if s.httpSrv != nil {
		log.Info("Waiting for frontend to be ready...(send POST /runsim to HTTP server)")
		//wait for the frontend to connect
		select {
		case <-s.runC:
		case <-ctx.Done():
			return Result{
				Duration: time.Since(start),
				Error:    ctx.Err(),
			}
		}
		log.Info("Received signal from frontend - starting simulation run.")
	}
	errc := make(chan error)
	quit := make(chan struct{})
	defer close(quit)
	go func() {
		select {
		case errc <- f(ctx, s):
		case <-quit:
		}
	}()
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errc:
	}
	return Result{
		Duration: time.Since(start),
		Error:    err,
	}
}

// Maximal number of parallel calls to cleanup functions on
// Simulation.Close.
var maxParallelCleanups = 10

// Close calls all cleanup functions that are returned by
// ServiceFunc, waits for all of them to finish and other
// functions that explicitly block shutdownWG
// (like Simulation.PeerEvents) and shuts down the network
// at the end. It is used to clean all resources from the
// simulation.
func (s *Simulation) Close() {
	close(s.done)

	sem := make(chan struct{}, maxParallelCleanups)
	s.mu.RLock()
	cleanupFuncs := make([]func(), len(s.cleanupFuncs))
	for i, f := range s.cleanupFuncs {
		if f != nil {
			cleanupFuncs[i] = f
		}
	}
	s.mu.RUnlock()
	var cleanupWG sync.WaitGroup
	for _, cleanup := range cleanupFuncs {
		cleanupWG.Add(1)
		sem <- struct{}{}
		go func(cleanup func()) {
			defer cleanupWG.Done()
			defer func() { <-sem }()

			cleanup()
		}(cleanup)
	}
	cleanupWG.Wait()

	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := s.httpSrv.Shutdown(ctx)
		if err != nil {
			log.Error("Error shutting down HTTP server!", "err", err)
		}
		close(s.runC)
	}

	s.shutdownWG.Wait()
	s.Net.Shutdown()
}

// Done returns a channel that is closed when the simulation
// is closed by Close method. It is useful for signaling termination
// of all possible goroutines that are created within the test.
func (s *Simulation) Done() <-chan struct{} {
	return s.done
}
