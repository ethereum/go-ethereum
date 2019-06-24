goupnp is a UPnP client library for Go

Installation
------------

Run `go get -u github.com/huin/goupnp`.

Documentation
-------------

Supported DCPs (you probably want to start with one of these):

* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) av1](https://godoc.org/github.com/huin/goupnp/dcps/av1) - Client for UPnP Device Control Protocol MediaServer v1 and MediaRenderer v1.
* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) internetgateway1](https://godoc.org/github.com/huin/goupnp/dcps/internetgateway1) - Client for UPnP Device Control Protocol Internet Gateway Device v1.
* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) internetgateway2](https://godoc.org/github.com/huin/goupnp/dcps/internetgateway2) - Client for UPnP Device Control Protocol Internet Gateway Device v2.

Core components:

* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) (goupnp)](https://godoc.org/github.com/huin/goupnp) core library - contains datastructures and utilities typically used by the implemented DCPs.
* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) httpu](https://godoc.org/github.com/huin/goupnp/httpu) HTTPU implementation, underlies SSDP.
* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) ssdp](https://godoc.org/github.com/huin/goupnp/ssdp) SSDP client implementation (simple service discovery protocol) - used to discover UPnP services on a network.
* [![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg) soap](https://godoc.org/github.com/huin/goupnp/soap) SOAP client implementation (simple object access protocol) - used to communicate with discovered services.


Regenerating dcps generated source code:
----------------------------------------

1. Build code generator:

	`go get -u github.com/huin/goupnp/cmd/goupnpdcpgen`

2. Regenerate the code:

	`go generate ./...`

Supporting additional UPnP devices and services:
------------------------------------------------

Supporting additional services is, in the trivial case, simply a matter of
adding the service to the `dcpMetadata` whitelist in `cmd/goupnpdcpgen/metadata.go`,
regenerating the source code (see above), and committing that source code.

However, it would be helpful if anyone needing such a service could test the
service against the service they have, and then reporting any trouble
encountered as an [issue on this
project](https://github.com/huin/goupnp/issues/new). If it just works, then
please report at least minimal working functionality as an issue, and
optionally contribute the metadata upstream.
