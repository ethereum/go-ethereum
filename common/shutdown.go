// Copyright 2014 The go-ethereum Authors
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

package common

import (
	"sync"
)

// ShutdownManager implements a sync mechanism that waits for multiple jobs
// to finish and also prevents starting new jobs after the shutdown process
// has started.
type ShutdownManager struct {
	wg sync.WaitGroup
	lock sync.RWMutex
	stop bool
	stopChn chan struct{}
}

// NewShutdownManager creates a new ShutdownManager instance
func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{stopChn: make(chan struct{})}
}

// Enter should be called before beginning a new job. If Enter returns false,
// the shutdown process has already been started. In this case, Exit should not
// be called.
func (self *ShutdownManager) Enter() bool {
	self.lock.RLock()
	defer self.lock.RUnlock()

	if self.stop {
		return false
	}
	self.wg.Add(1)
	return true
}

// Exit should be called after finishing a job
func (self *ShutdownManager) Exit() {
	self.wg.Done()
}

// Shutdown initiates the shutdown process. After calling it, any subsequent
// calls to Enter will return false. Shutdown returns after all active jobs
// have finished and called Exit.
func (self *ShutdownManager) Shutdown() {
	self.lock.Lock()
	self.stop = true
	self.lock.Unlock()
	close(self.stopChn)
	self.wg.Wait()
}

// Stopped returns true if the shutdown process has already been started
func (self *ShutdownManager) Stopped() bool {
	self.lock.RLock()
	defer self.lock.RUnlock()
	
	return self.stop
}

// StopChannel returns a channel which is closed when the shutdown starts
func (self *ShutdownManager) StopChannel() chan struct{} {
	return self.stopChn
}