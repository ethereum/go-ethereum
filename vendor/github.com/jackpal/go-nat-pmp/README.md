go-nat-pmp
==========

A Go language client for the NAT-PMP internet protocol for port mapping and discovering the external
IP address of a firewall.

NAT-PMP is supported by Apple brand routers and open source routers like Tomato and DD-WRT.

See https://tools.ietf.org/rfc/rfc6886.txt


[![Build Status](https://travis-ci.org/jackpal/go-nat-pmp.svg)](https://travis-ci.org/jackpal/go-nat-pmp)

Get the package
---------------

    go get -u github.com/jackpal/go-nat-pmp

Usage
-----

    import (
        "fmt"
        "github.com/jackpal/gateway"
        natpmp "github.com/jackpal/go-nat-pmp"
    )

    gatewayIP, err := gateway.DiscoverGateway()
    if err != nil {
        return
    }

    client := natpmp.NewClient(gatewayIP)
    response, err := client.GetExternalAddress()
    if err != nil {
        return
    }
    fmt.Println("External IP address: %v", response.ExternalIPAddress)

Clients
-------

This library is used in the Taipei Torrent BitTorrent client http://github.com/jackpal/Taipei-Torrent

Complete documentation
----------------------

    http://godoc.org/github.com/jackpal/go-nat-pmp

License
-------

This project is licensed under the Apache License 2.0.
