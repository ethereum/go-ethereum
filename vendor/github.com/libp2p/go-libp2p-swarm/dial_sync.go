package swarm

import (
	"context"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
)

type DialFunc func(context.Context, peer.ID) (*Conn, error)

func NewDialSync(dfn DialFunc) *DialSync {
	return &DialSync{
		dials:    make(map[peer.ID]*activeDial),
		dialFunc: dfn,
	}
}

type DialSync struct {
	dials    map[peer.ID]*activeDial
	dialsLk  sync.Mutex
	dialFunc DialFunc
}

type activeDial struct {
	id       peer.ID
	refCnt   int
	refCntLk sync.Mutex
	cancel   func()

	err    error
	conn   *Conn
	waitch chan struct{}

	ds *DialSync
}

func (dr *activeDial) wait(ctx context.Context) (*Conn, error) {
	defer dr.decref()
	select {
	case <-dr.waitch:
		return dr.conn, dr.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (ad *activeDial) incref() {
	ad.refCntLk.Lock()
	defer ad.refCntLk.Unlock()
	ad.refCnt++
}

func (ad *activeDial) decref() {
	ad.refCntLk.Lock()
	ad.refCnt--
	maybeZero := (ad.refCnt <= 0)
	ad.refCntLk.Unlock()

	// make sure to always take locks in correct order.
	if maybeZero {
		ad.ds.dialsLk.Lock()
		ad.refCntLk.Lock()
		// check again after lock swap drop to make sure nobody else called incref
		// in between locks
		if ad.refCnt <= 0 {
			ad.cancel()
			delete(ad.ds.dials, ad.id)
		}
		ad.refCntLk.Unlock()
		ad.ds.dialsLk.Unlock()
	}
}

func (ad *activeDial) start(ctx context.Context) {
	ad.conn, ad.err = ad.ds.dialFunc(ctx, ad.id)
	close(ad.waitch)
	ad.cancel()
}

func (ds *DialSync) getActiveDial(p peer.ID) *activeDial {
	ds.dialsLk.Lock()
	defer ds.dialsLk.Unlock()

	actd, ok := ds.dials[p]
	if !ok {
		adctx, cancel := context.WithCancel(context.Background())
		actd = &activeDial{
			id:     p,
			cancel: cancel,
			waitch: make(chan struct{}),
			ds:     ds,
		}
		ds.dials[p] = actd

		go actd.start(adctx)
	}

	// increase ref count before dropping dialsLk
	actd.incref()

	return actd
}

func (ds *DialSync) DialLock(ctx context.Context, p peer.ID) (*Conn, error) {
	return ds.getActiveDial(p).wait(ctx)
}
