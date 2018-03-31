/*
Package multiaddr provides an implementation of the Multiaddr network
address format. Multiaddr emphasizes explicitness, self-description, and
portability. It allows applications to treat addresses as opaque tokens,
and to avoid making assumptions about the address representation (e.g. length).
Learn more at https://github.com/multiformats/multiaddr

Basic Use:

  import (
    "bytes"
    "strings"
    ma "github.com/multiformats/go-multiaddr"
  )

  // construct from a string (err signals parse failure)
  m1, err := ma.NewMultiaddr("/ip4/127.0.0.1/udp/1234")

  // construct from bytes (err signals parse failure)
  m2, err := ma.NewMultiaddrBytes(m1.Bytes())

  // true
  strings.Equal(m1.String(), "/ip4/127.0.0.1/udp/1234")
  strings.Equal(m1.String(), m2.String())
  bytes.Equal(m1.Bytes(), m2.Bytes())
  m1.Equal(m2)
  m2.Equal(m1)

  // tunneling (en/decap)
  printer, _ := ma.NewMultiaddr("/ip4/192.168.0.13/tcp/80")
  proxy, _ := ma.NewMultiaddr("/ip4/10.20.30.40/tcp/443")
  printerOverProxy := proxy.Encapsulate(printer)
  proxyAgain := printerOverProxy.Decapsulate(printer)

*/
package multiaddr
