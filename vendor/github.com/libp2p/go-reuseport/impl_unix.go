// +build darwin freebsd dragonfly netbsd openbsd linux

package reuseport

import (
	"context"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/libp2p/go-reuseport/singlepoll"
	sockaddrnet "github.com/libp2p/go-sockaddr/net"
)

const (
	filePrefix = "port."
)

// Wrapper around the socket system call that marks the returned file
// descriptor as nonblocking and close-on-exec.
func socket(family, socktype, protocol int) (fd int, err error) {
	syscall.ForkLock.RLock()
	fd, err = unix.Socket(family, socktype, protocol)
	if err == nil {
		unix.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()

	if err != nil {
		return -1, err
	}

	// cant set it until after connect
	// if err = unix.SetNonblock(fd, true); err != nil {
	// 	unix.Close(fd)
	// 	return -1, err
	// }

	if err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		unix.Close(fd)
		return -1, err
	}

	if err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		unix.Close(fd)
		return -1, err
	}

	// set setLinger to 5 as reusing exact same (srcip:srcport, dstip:dstport)
	// will otherwise fail on connect.
	if err = setLinger(fd, 5); err != nil {
		unix.Close(fd)
		return -1, err
	}

	return fd, nil
}

func dial(ctx context.Context, dialer net.Dialer, netw, addr string) (c net.Conn, err error) {
	var (
		fd             int
		lfamily        int
		rfamily        int
		socktype       int
		lprotocol      int
		rprotocol      int
		file           *os.File
		deadline       time.Time
		remoteSockaddr unix.Sockaddr
		localSockaddr  unix.Sockaddr
	)

	netAddr, err := ResolveAddr(netw, addr)
	if err != nil {
		return nil, err
	}

	switch netAddr.(type) {
	case *net.TCPAddr, *net.UDPAddr:
	default:
		return nil, ErrUnsupportedProtocol
	}

	switch {
	case !dialer.Deadline.IsZero():
		deadline = dialer.Deadline
	case dialer.Timeout != 0:
		deadline = time.Now().Add(dialer.Timeout)
	}

	ctxdeadline, ok := ctx.Deadline()
	if ok && ctxdeadline.Before(deadline) {
		deadline = ctxdeadline
	}

	localSockaddr = sockaddrnet.NetAddrToSockaddr(dialer.LocalAddr)
	remoteSockaddr = sockaddrnet.NetAddrToSockaddr(netAddr)

	rfamily = sockaddrnet.NetAddrAF(netAddr)
	rprotocol = sockaddrnet.NetAddrIPPROTO(netAddr)
	socktype = sockaddrnet.NetAddrSOCK(netAddr)

	if dialer.LocalAddr != nil {
		switch dialer.LocalAddr.(type) {
		case *net.TCPAddr, *net.UDPAddr:
		default:
			return nil, ErrUnsupportedProtocol
		}

		// check family and protocols match.
		lfamily = sockaddrnet.NetAddrAF(dialer.LocalAddr)
		lprotocol = sockaddrnet.NetAddrIPPROTO(dialer.LocalAddr)
		if lfamily != rfamily || lprotocol != rprotocol {
			return nil, &net.AddrError{Err: "unexpected address type", Addr: netAddr.String()}
		}
	}

	// look at dialTCP in http://golang.org/src/net/tcpsock_posix.go  .... !
	// here we just try again 3 times.
	for i := 0; i < 3; i++ {
		if !deadline.IsZero() && deadline.Before(time.Now()) {
			err = errTimeout
			break
		}

		if fd, err = socket(rfamily, socktype, rprotocol); err != nil {
			return nil, err
		}

		if localSockaddr != nil {
			if err = unix.Bind(fd, localSockaddr); err != nil {
				unix.Close(fd)
				return nil, err
			}
		}

		if err = unix.SetNonblock(fd, true); err != nil {
			unix.Close(fd)
			return nil, err
		}

		if err = connect(ctx, fd, remoteSockaddr, deadline); err != nil {
			unix.Close(fd)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue // try again.
		}

		break
	}
	if err != nil {
		return nil, err
	}

	if rprotocol == unix.IPPROTO_TCP {
		sa, err := unix.Getsockname(fd)
		if err != nil {
			unix.Close(fd)
			return nil, err
		}
		ra, err := unix.Getpeername(fd)
		if err != nil {
			unix.Close(fd)
			return nil, err
		}

		// Need to call setLinger 0. Otherwise, the close will block for the linger period. Linux bug?
		switch s := sa.(type) {
		case *unix.SockaddrInet4:
			if r, ok := ra.(*unix.SockaddrInet4); ok && r.Addr == s.Addr && r.Port == s.Port {
				setLinger(fd, 0)
				unix.Close(fd)
				return nil, ErrDialSelf
			}
		case *unix.SockaddrInet6:
			if r, ok := ra.(*unix.SockaddrInet6); ok && r.Addr == s.Addr && r.Port == s.Port && r.ZoneId == s.ZoneId {
				setLinger(fd, 0)
				unix.Close(fd)
				return nil, ErrDialSelf
			}
		}

		//  by default golang/net sets TCP no delay to true.
		if err = setNoDelay(fd, true); err != nil {
			unix.Close(fd)
			return nil, err
		}
	}

	// NOTE:XXX: never call unix.Close on fd after os.NewFile
	file = os.NewFile(uintptr(fd), filePrefix+strconv.Itoa(os.Getpid()))
	fd = -1 // so we don't touch it, we handled the control to Golang with NewFile
	if c, err = net.FileConn(file); err != nil {
		_ = file.Close() // shouldn't error either way
		return nil, err
	}

	if err = file.Close(); err != nil {
		c.Close()
		return nil, err
	}

	// c = wrapConnWithRemoteAddr(c, netAddr)
	return c, err
}

func listen(netw, addr string) (fd int, err error) {
	var (
		family   int
		socktype int
		protocol int
		sockaddr unix.Sockaddr
	)

	netAddr, err := ResolveAddr(netw, addr)
	if err != nil {
		return -1, err
	}

	switch netAddr.(type) {
	case *net.TCPAddr, *net.UDPAddr:
	default:
		return -1, ErrUnsupportedProtocol
	}

	family = sockaddrnet.NetAddrAF(netAddr)
	protocol = sockaddrnet.NetAddrIPPROTO(netAddr)
	sockaddr = sockaddrnet.NetAddrToSockaddr(netAddr)
	socktype = sockaddrnet.NetAddrSOCK(netAddr)

	if fd, err = socket(family, socktype, protocol); err != nil {
		return -1, err
	}

	if err = unix.Bind(fd, sockaddr); err != nil {
		unix.Close(fd)
		return -1, err
	}

	if protocol == unix.IPPROTO_TCP {
		//  by default golang/net sets TCP no delay to true.
		if err = setNoDelay(fd, true); err != nil {
			unix.Close(fd)
			return -1, err
		}
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return -1, err
	}

	return fd, nil
}

func listenStream(netw, addr string) (l net.Listener, err error) {
	var (
		file *os.File
	)

	fd, err := listen(netw, addr)
	if err != nil {
		return nil, err
	}

	// Set backlog size to the maximum
	if err = unix.Listen(fd, unix.SOMAXCONN); err != nil {
		unix.Close(fd)
		return nil, err
	}

	file = os.NewFile(uintptr(fd), filePrefix+strconv.Itoa(os.Getpid()))
	if l, err = net.FileListener(file); err != nil {
		unix.Close(fd)
		return nil, err
	}

	if err = file.Close(); err != nil {
		unix.Close(fd)
		l.Close()
		return nil, err
	}

	return l, err
}

func listenPacket(netw, addr string) (p net.PacketConn, err error) {
	var (
		file *os.File
	)

	fd, err := listen(netw, addr)
	if err != nil {
		return nil, err
	}

	file = os.NewFile(uintptr(fd), filePrefix+strconv.Itoa(os.Getpid()))
	if p, err = net.FilePacketConn(file); err != nil {
		unix.Close(fd)
		return nil, err
	}

	if err = file.Close(); err != nil {
		unix.Close(fd)
		p.Close()
		return nil, err
	}

	return p, err
}

func listenUDP(netw, addr string) (c net.Conn, err error) {
	var (
		file *os.File
	)

	fd, err := listen(netw, addr)
	if err != nil {
		return nil, err
	}

	file = os.NewFile(uintptr(fd), filePrefix+strconv.Itoa(os.Getpid()))
	if c, err = net.FileConn(file); err != nil {
		unix.Close(fd)
		return nil, err
	}

	if err = file.Close(); err != nil {
		unix.Close(fd)
		return nil, err
	}

	return c, err
}

// this is close to the connect() function inside stdlib/net
func connect(ctx context.Context, fd int, ra unix.Sockaddr, deadline time.Time) error {
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	switch err := unix.Connect(fd, ra); err {
	case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
	case nil, unix.EISCONN:
		if !deadline.IsZero() && deadline.Before(time.Now()) {
			return errTimeout
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	default:
		return err
	}

	for {
		if err := singlepoll.PollPark(ctx, fd, "w"); err != nil {
			return err
		}

		// if err := fd.pd.WaitWrite(); err != nil {
		// 	return err
		// }
		// i'd use the above fd.pd.WaitWrite to poll io correctly, just like net sockets...
		// but of course, it uses the damn runtime_* functions that _cannot_ be used by
		// non-go-stdlib source... seriously guys, this is not nice.
		// we're relegated to using unix.Select (what nightmare that is) or using
		// a simple but totally bogus time-based wait. such garbage.
		nerr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
		if err != nil {
			return err
		}
		switch err = syscall.Errno(nerr); err {
		case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
			continue
		case syscall.Errno(0), unix.EISCONN:
			if !deadline.IsZero() && deadline.Before(time.Now()) {
				return errTimeout
			}
			return nil
		default:
			return err
		}
	}
}

var errTimeout = &timeoutError{}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
