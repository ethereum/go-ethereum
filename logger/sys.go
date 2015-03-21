package logger

import (
	"sync"
)

type message struct {
	level LogLevel
	msg   string
}

var (
	logMessageC = make(chan message)
	addSystemC  = make(chan LogSystem)
	flushC      = make(chan chan struct{})
	resetC      = make(chan chan struct{})
)

func init() {
	go dispatchLoop()
}

// each system can buffer this many messages before
// blocking incoming log messages.
const sysBufferSize = 500

func dispatchLoop() {
	var (
		systems  []LogSystem
		systemIn []chan message
		systemWG sync.WaitGroup
	)
	bootSystem := func(sys LogSystem) {
		in := make(chan message, sysBufferSize)
		systemIn = append(systemIn, in)
		systemWG.Add(1)
		go sysLoop(sys, in, &systemWG)
	}

	for {
		select {
		case msg := <-logMessageC:
			for _, c := range systemIn {
				c <- msg
			}

		case sys := <-addSystemC:
			systems = append(systems, sys)
			bootSystem(sys)

		case waiter := <-resetC:
			// reset means terminate all systems
			for _, c := range systemIn {
				close(c)
			}
			systems = nil
			systemIn = nil
			systemWG.Wait()
			close(waiter)

		case waiter := <-flushC:
			// flush means reboot all systems
			for _, c := range systemIn {
				close(c)
			}
			systemIn = nil
			systemWG.Wait()
			for _, sys := range systems {
				bootSystem(sys)
			}
			close(waiter)
		}
	}
}

func sysLoop(sys LogSystem, in <-chan message, wg *sync.WaitGroup) {
	for msg := range in {
		switch sys.(type) {
		case *jsonLogSystem:
			if msg.level == JsonLevel {
				sys.LogPrint(msg.level, msg.msg)
			}
		default:
			if sys.GetLogLevel() >= msg.level {
				sys.LogPrint(msg.level, msg.msg)
			}
		}
	}
	wg.Done()
}

// Reset removes all active log systems.
// It blocks until all current messages have been delivered.
func Reset() {
	waiter := make(chan struct{})
	resetC <- waiter
	<-waiter
}

// Flush waits until all current log messages have been dispatched to
// the active log systems.
func Flush() {
	waiter := make(chan struct{})
	flushC <- waiter
	<-waiter
}

// AddLogSystem starts printing messages to the given LogSystem.
func AddLogSystem(sys LogSystem) {
	addSystemC <- sys
}
