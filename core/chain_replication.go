package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type ChainReplicationBackend interface {
	Process(ctx context.Context, events []*BlockReplicationEvent) error
	String() string
}

// ChainReplicationChain interface is used for connecting the replicator to a blockchain
type ChainReplicatorChain interface {
	// SubscribeChainReplicationEvent subscribes to new replication notifications.
	SubscribeBlockReplicationEvent(ch chan<- BlockReplicationEvent) event.Subscription
	// Set Block replica export types
	SetBlockReplicaExports(replicaConfig *ReplicaConfig) bool
}

type ChainReplicator struct {
	sessionId uint64

	backend ChainReplicationBackend

	mode       uint32
	modeLock   sync.Mutex
	drain      chan struct{}
	exitStatus chan error
	ctx        context.Context
	ctxCancel  func()

	log *replicationLogger
}

var replicationSessionSeq uint64

func NewChainReplicator(backend ChainReplicationBackend) *ChainReplicator {
	sessionId := atomic.AddUint64(&replicationSessionSeq, 1)

	c := &ChainReplicator{
		sessionId:  sessionId,
		backend:    backend,
		drain:      make(chan struct{}),
		exitStatus: make(chan error),
		log:        &replicationLogger{log: log.New("sessID", sessionId)},
	}

	c.ctx, c.ctxCancel = context.WithCancel(context.Background())

	return c
}

const (
	modeNotStarted uint32 = iota
	modeStarting
	modeRunning
	modeStopping
)

func (c *ChainReplicator) Start(chain ChainReplicatorChain, replicaConfig *ReplicaConfig) {
	c.modeLock.Lock()
	defer c.modeLock.Unlock()

	if !atomic.CompareAndSwapUint32(&c.mode, modeNotStarted, modeStarting) {
		return
	}

	c.log.Info("Replication began", "backend", c.backend.String())

	bSEvents := make(chan BlockReplicationEvent, 1000)
	bSSub := chain.SubscribeBlockReplicationEvent(bSEvents)
	_ = chain.SetBlockReplicaExports(replicaConfig)
	go c.eventLoop(bSEvents, bSSub)
}

func (c *ChainReplicator) Stop() (err error) {
	c.modeLock.Lock()
	defer c.modeLock.Unlock()

	if !atomic.CompareAndSwapUint32(&c.mode, modeRunning, modeStopping) {
		return
	}

	close(c.drain)
	err = <-c.exitStatus
	atomic.StoreUint32(&c.mode, modeNotStarted)

	return
}

func (c *ChainReplicator) CloseImmediate() (err error) {
	if atomic.LoadUint32(&c.mode) == modeStopping {
		// Stop() or another CloseImmediate() is already holding the lock,
		// so just hurry it along (idempotently)
		c.ctxCancel()

		// wait for the other task to finish
		c.modeLock.Lock()
		defer c.modeLock.Unlock()

		return
	}

	c.modeLock.Lock()
	defer c.modeLock.Unlock()

	if !atomic.CompareAndSwapUint32(&c.mode, modeRunning, modeStopping) {
		return
	}

	c.ctxCancel()
	err = <-c.exitStatus
	atomic.StoreUint32(&c.mode, modeNotStarted)

	return
}

var (
	errComplete     = errors.New("replication complete")
	errDraining     = errors.New("draining")
	errUnsubscribed = errors.New("unsubscribed")
	errContextDone  = errors.New("context completed")
)

func (c *ChainReplicator) eventLoop(events chan BlockReplicationEvent, sub event.Subscription) {
	defer sub.Unsubscribe()
	defer close(c.exitStatus)
	defer c.log.nextSession()

	atomic.StoreUint32(&c.mode, modeRunning)

	var (
		draining bool
		unsubbed bool

		flush          bool
		lastFlushTime  = time.Now()
		lastReportTime = lastFlushTime

		ticker   = time.NewTicker(1 * time.Second)
		eventBuf = make([]*BlockReplicationEvent, 0, 500)

		stateChange = make(chan error, 2)
		loopDone    = make(chan struct{})
	)

	defer ticker.Stop()
	defer close(loopDone)

	go func() {
		select {
		case err, ok := <-sub.Err():
			if ok {
				stateChange <- err
			} else {
				stateChange <- errUnsubscribed
			}
			return
		case <-loopDone:
			return
		}
	}()

	go func() {
		select {
		case <-c.ctx.Done():
			stateChange <- errContextDone
			return
		case <-loopDone:
			return
		}
	}()

	go func() {
		select {
		case <-c.drain:
			stateChange <- errDraining
		case <-loopDone:
			return
		}
	}()

	for {
		select {
		case err := <-stateChange:
			switch err {
			case errComplete:
				c.log.Info("Replication complete")
				return

			case errDraining:
				if !draining {
					c.log.Info("Replication queue draining")
					draining = true
				}

			case errUnsubscribed:
				if !unsubbed {
					c.log.Debug("Replication producer unsubscribed")
					unsubbed = true
					flush = true
				}

			case errContextDone:
				c.log.Info("Replication interrupted")
				return

			default:
				// a real error, from the subscription producer
				c.log.Warn("Replication failure", "err", err)
				c.exitStatus <- err
				return
			}

		case ev, ok := <-events:
			if ok {
				eventBuf = append(eventBuf, &ev)

				if len(eventBuf) == 500 {
					flush = true
				}
			} else {
				stateChange <- errUnsubscribed
			}

		case t := <-ticker.C:
			if t.Sub(lastReportTime) >= (8 * time.Second) {
				if len(eventBuf) > 0 || c.log.IsDirty() {
					c.log.Info("Replication progress", "queued", len(eventBuf))
				}
				lastReportTime = t
			}

			if len(eventBuf) > 0 && t.Sub(lastFlushTime) >= (3*time.Second) {
				flush = true
			} else if draining && len(eventBuf) == 0 && t.Sub(lastFlushTime) >= (2*time.Second) {
				c.log.Info("Replication complete (queue drained)")
				return
			}
		}

		if flush {
			if len(eventBuf) > 0 {
				if err := c.backend.Process(c.ctx, eventBuf); err != nil {
					stateChange <- err
				} else {
					c.log.sent(eventBuf)
					c.log.Debug("Replication segment", "len", len(eventBuf))
				}
			}

			flush = false
			lastFlushTime = time.Now()
			eventBuf = eventBuf[:0]

			if unsubbed {
				stateChange <- errComplete
				unsubbed = false
			}
		}
	}
}

type replicationLogger struct {
	log          log.Logger
	flushedCount uint64
	hashValid    bool
	lastHash     string
	dirty        bool
}

func (rl *replicationLogger) IsDirty() bool {
	return rl.dirty
}

func (rl *replicationLogger) appendMetrics(input []interface{}) []interface{} {
	rl.dirty = false

	if rl.hashValid {
		return append(input, "sent", rl.flushedCount, "last", rl.lastHash)
	} else {
		return append(input, "sent", rl.flushedCount)
	}
}

func (rl *replicationLogger) nextSession() {
	rl.hashValid = false
	rl.dirty = false
}

func (rl *replicationLogger) sent(eventBuf []*BlockReplicationEvent) {
	if len(eventBuf) == 0 {
		return
	}

	rl.flushedCount += uint64(len(eventBuf))
	rl.lastHash = eventBuf[len(eventBuf)-1].Hash
	rl.hashValid = true
	rl.dirty = true
}

func (rl *replicationLogger) Trace(slug string, ctx ...interface{}) {
	rl.log.Trace(slug, rl.appendMetrics(ctx)...)
}
func (rl *replicationLogger) Debug(slug string, ctx ...interface{}) {
	rl.log.Debug(slug, rl.appendMetrics(ctx)...)
}
func (rl *replicationLogger) Info(slug string, ctx ...interface{}) {
	rl.log.Info(slug, rl.appendMetrics(ctx)...)
}
func (rl *replicationLogger) Warn(slug string, ctx ...interface{}) {
	rl.log.Warn(slug, rl.appendMetrics(ctx)...)
}
func (rl *replicationLogger) Error(slug string, ctx ...interface{}) {
	rl.log.Error(slug, rl.appendMetrics(ctx)...)
}
func (rl *replicationLogger) Crit(slug string, ctx ...interface{}) {
	rl.log.Crit(slug, rl.appendMetrics(ctx)...)
}
