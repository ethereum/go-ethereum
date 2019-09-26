# ECDH

[![Build Status](https://travis-ci.org/wsddn/go-ecdh.svg?branch=master)](https://travis-ci.org/wsddn/go-ecdh)

This is a go implementation of elliptical curve diffie-hellman key exchange method.
It supports the NIST curves (and any curves using the `elliptic.Curve` go interface)
as well as djb's curve25519. 

The library handles generating of keys, generating a shared secret, and the
(un)marshalling of the elliptical curve keys into slices of bytes.

## Warning and Disclaimer
I am not a cryptographer, this was written as part of a personal project to learn about cryptographic systems and protocols. No claims as to the security of this library are made, I would not advise using it for anything that requires any level of security. Pull requests or issues about security flaws are however still welcome.

## Compatibility
Works with go 1.2 onwards.

## TODO
 * Improve documentation
