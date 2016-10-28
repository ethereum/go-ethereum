goupnp is a UPnP client library for Go

Installation
------------

Run `go get -u github.com/huin/goupnp`.

Documentation
-------------

All doc links below are for ![GoDoc](https://godoc.org/github.com/huin/goupnp?status.svg).

Supported DCPs (you probably want to start with one of these):
* [av1](https://godoc.org/github.com/huin/goupnp/dcps/av1) - Client for UPnP Device Control Protocol MediaServer v1 and MediaRenderer v1.
* [internetgateway1](https://godoc.org/github.com/huin/goupnp/dcps/internetgateway1) - Client for UPnP Device Control Protocol Internet Gateway Device v1.
* [internetgateway2](https://godoc.org/github.com/huin/goupnp/dcps/internetgateway2) - Client for UPnP Device Control Protocol Internet Gateway Device v2.

Core components:
* [(goupnp)](https://godoc.org/github.com/huin/goupnp) core library - contains datastructures and utilities typically used by the implemented DCPs.
* [httpu](https://godoc.org/github.com/huin/goupnp/httpu) HTTPU implementation, underlies SSDP.
* [ssdp](https://godoc.org/github.com/huin/goupnp/ssdp) SSDP client implementation (simple service discovery protocol) - used to discover UPnP services on a network.
* [soap](https://godoc.org/github.com/huin/goupnp/soap) SOAP client implementation (simple object access protocol) - used to communicate with discovered services.


Regenerating dcps generated source code:
----------------------------------------

1. Install gotasks: `go get -u github.com/jingweno/gotask`
2. Change to the gotasks directory: `cd gotasks`
3. Run specgen task: `gotask specgen`

Supporting additional UPnP devices and services:
------------------------------------------------

Supporting additional services is, in the trivial case, simply a matter of
adding the service to the `dcpMetadata` whitelist in `gotasks/specgen_task.go`,
regenerating the source code (see above), and committing that source code.

However, it would be helpful if anyone needing such a service could test the
service against the service they have, and then reporting any trouble
encountered as an [issue on this
project](https://github.com/huin/goupnp/issues/new). If it just works, then
please report at least minimal working functionality as an issue, and
optionally contribute the metadata upstream.
