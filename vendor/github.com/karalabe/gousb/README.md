Introduction
============

[![Travis Build Status][travisimg]][travis]
[![AppVeyor Build Status][appveyorimg]][appveyor]
[![GoDoc][docimg]][doc]

The gousb package is an attempt at wrapping the `libusb` library into a Go-like binding in a fully self-contained, go-gettable package. Supported platforms include Linux, macOS and Windows as well as the mobile platforms Android and iOS.

This package is a fork of [`github.com/kylelemons/gousb`](https://github.com/kylelemons/gousb), which at the moment seems to be unmaintained. The current fork is different from the upstream package as it contains code to embed `libusb` directly into the Go package (thus becoming fully self-cotnained and go-gettable), as well as it features a few contributions and bugfixes that never really got addressed in the upstream package, but which address important issues nonetheless.

*Note, if @kylelemons decides to pick development of the upstream project up again, consider all commits made by me to this repo as ready contributions. I cannot vouch for other commits as the upstream repo needs a signed CLA for Google.*

[travisimg]:   https://travis-ci.org/karalabe/gousb.svg?branch=master
[travis]:      https://travis-ci.org/karalabe/gousb
[appveyorimg]: https://ci.appveyor.com/api/projects/status/84k9xse10rl72gn2/branch/master?svg=true
[appveyor]:    https://ci.appveyor.com/project/karalabe/gousb
[docimg]:      https://godoc.org/github.com/karalabe/gousb?status.svg
[doc]:         https://godoc.org/github.com/karalabe/gousb

Installation
============

Example: lsusb
--------------
The gousb project provides a simple but useful example: lsusb.  This binary will list the USB devices connected to your system and various interesting tidbits about them, their configurations, endpoints, etc.  To install it, run the following command:

    go get -v github.com/karalabe/gousb/lsusb

gousb
-----
If you installed the lsusb example, both libraries below are already installed.

Installing the primary gousb package is really easy:

    go get -v github.com/karalabe/gousb/usb

There is also a `usbid` package that will not be installed by default by this command, but which provides useful information including the human-readable vendor and product codes for detected hardware.  It's not installed by default and not linked into the `usb` package by default because it adds ~400kb to the resulting binary.  If you want both, they can be installed thus:

    go get -v github.com/karalabe/gousb/usb{,id}

Documentation
=============
The documentation can be viewed via local godoc or via the excellent [godoc.org](http://godoc.org/):

- [usb](http://godoc.org/github.com/karalabe/gousb/usb)
- [usbid](http://godoc.org/pkg/github.com/karalabe/gousb/usbid)
