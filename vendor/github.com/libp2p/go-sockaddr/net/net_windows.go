package sockaddrnet

import (
	"golang.org/x/sys/windows"
)

const (
	AF_INET   = windows.AF_INET
	AF_INET6  = windows.AF_INET6
	AF_UNIX   = windows.AF_UNIX
	AF_UNSPEC = windows.AF_UNSPEC

	IPPROTO_IP   = windows.IPPROTO_IP
	IPPROTO_IPV4 = 0x4 // windows.IPPROTO_IPV4 (missing)
	IPPROTO_IPV6 = windows.IPPROTO_IPV6
	IPPROTO_TCP  = windows.IPPROTO_TCP
	IPPROTO_UDP  = windows.IPPROTO_UDP

	SOCK_DGRAM     = windows.SOCK_DGRAM
	SOCK_STREAM    = windows.SOCK_STREAM
	SOCK_SEQPACKET = windows.SOCK_SEQPACKET
)

type Sockaddr = windows.Sockaddr
type SockaddrInet4 = windows.SockaddrInet4
type SockaddrInet6 = windows.SockaddrInet6
type SockaddrUnix = windows.SockaddrUnix
type RawSockaddrAny = windows.RawSockaddrAny
