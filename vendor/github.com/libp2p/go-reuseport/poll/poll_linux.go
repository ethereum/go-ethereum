// +build linux

package poll

import (
	"context"
	"golang.org/x/sys/unix"
	"sync"

	"github.com/gxed/eventfd"
)

type Poller struct {
	epfd int

	eventMain unix.EpollEvent
	eventWait unix.EpollEvent
	events    []unix.EpollEvent

	wake      *eventfd.EventFD // Use eventfd to wakeup epoll
	wakeMutex sync.Mutex
}

func New(fd int) (p *Poller, err error) {
	p = &Poller{
		events: make([]unix.EpollEvent, 32),
	}
	if p.epfd, err = unix.EpollCreate1(0); err != nil {
		return nil, err
	}
	wake, err := eventfd.New()
	if err != nil {
		unix.Close(p.epfd)
		return nil, err
	}
	p.wake = wake

	p.eventMain.Events = unix.EPOLLOUT
	p.eventMain.Fd = int32(fd)
	if err = unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &p.eventMain); err != nil {
		p.Close()
		return nil, err
	}

	// poll that eventfd can be read
	p.eventWait.Events = unix.EPOLLIN
	p.eventWait.Fd = int32(wake.Fd())
	if err = unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, wake.Fd(), &p.eventWait); err != nil {
		p.wake.Close()
		p.Close()
		return nil, err
	}

	return p, nil
}

func (p *Poller) Close() error {
	p.wakeMutex.Lock()
	err1 := p.wake.Close()
	// set wake to nil to be sure that we won't call write on closed wake
	// it should never happen but if someone changes something this might show a bug
	p.wake = nil
	p.wakeMutex.Unlock()

	err2 := unix.Close(p.epfd)
	if err1 != nil {
		return err1
	} else {
		return err2
	}
}

func (p *Poller) WaitWriteCtx(ctx context.Context) error {
	doneChan := make(chan struct{})
	defer close(doneChan)

	go func() {
		select {
		case <-doneChan:
			return
		case <-ctx.Done():
			select {
			case <-doneChan:
				// if we re done with this function do not write to p.wake
				// it might be already closed and the fd could be reopened for
				// different purpose
				return
			default:
			}
			p.wakeMutex.Lock()
			if p.wake != nil {
				p.wake.WriteEvents(1) // send event to wake up epoll
			}
			// if it is nil then we already closed
			p.wakeMutex.Unlock()
			return
		}

	}()

	n, err := unix.EpollWait(p.epfd, p.events, -1)
	if err != nil {
		return err
	}
	good := false
	for i := 0; i < n; i++ {
		ev := p.events[i]
		switch ev.Fd {
		case p.eventMain.Fd:
			good = true
		case p.eventWait.Fd:
			p.wakeMutex.Lock()
			p.wake.ReadEvents() // clear eventfd
			p.wakeMutex.Unlock()
		default:
			// shouldn't happen as epoll should onlt return events we registered
		}
	}
	if good {
		// in case both eventMain and eventWait are lit, we got with eventMain
		// as it is the success condition here and if both of them are returned
		// at the same time it means that socket connected right as context timed out
		return nil
	}
	if ctx.Err() == nil {
		// notification is sent by other goroutine when context deadline was reached
		// if we are here it means that we got notification buy the deadline wasn't reached
		panic("notification but no deadline, this should be impossible")
	}
	return ctx.Err()
}
