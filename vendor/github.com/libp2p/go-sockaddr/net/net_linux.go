package sockaddrnet

import (
	"golang.org/x/sys/unix"
)

const (
	AF_INET   = unix.AF_INET
	AF_INET6  = unix.AF_INET6
	AF_UNIX   = unix.AF_UNIX
	AF_UNSPEC = unix.AF_UNSPEC

	IPPROTO_IP   = unix.IPPROTO_IP
	IPPROTO_IPV4 = unix.IPPROTO_IPIP
	IPPROTO_IPV6 = unix.IPPROTO_IPV6
	IPPROTO_TCP  = unix.IPPROTO_TCP
	IPPROTO_UDP  = unix.IPPROTO_UDP

	SOCK_DGRAM     = unix.SOCK_DGRAM
	SOCK_STREAM    = unix.SOCK_STREAM
	SOCK_SEQPACKET = unix.SOCK_SEQPACKET
)

type Sockaddr = unix.Sockaddr
type SockaddrInet4 = unix.SockaddrInet4
type SockaddrInet6 = unix.SockaddrInet6
type SockaddrUnix = unix.SockaddrUnix
type RawSockaddrAny = unix.RawSockaddrAny
