[![Travis][travisimg]][travisurl]
[![AppVeyor][appveyorimg]][appveyorurl]
[![GoDoc][docimg]][docurl]

[travisimg]:   https://travis-ci.org/karalabe/usb.svg?branch=master
[travisurl]:   https://travis-ci.org/karalabe/usb
[appveyorimg]: https://ci.appveyor.com/api/projects/status/u96eq262bj2itprh/branch/master?svg=true
[appveyorurl]: https://ci.appveyor.com/project/karalabe/usb
[docimg]:      https://godoc.org/github.com/karalabe/usb?status.svg
[docurl]:      https://godoc.org/github.com/karalabe/usb

# Yet another USB library for Go

The `usb` package is a cross platform, fully self-contained library for accessing and communicating with USB devices **either via HID or low level interrupts**. The goal of the library was to create a simple way to find-, attach to- and read/write form USB devices.

There are multiple already existing USB libraries:

 * The original `gousb` package [created by @kylelemons](https://github.com/kylelemons/gousb) and nowadays [maintained by @google](https://github.com/google/gousb) is a CGO wrapper around `libusb`. It is the most advanced USB library for Go out there.
   * Unfortunately, `gousb` requires the `libusb` C library to be installed both during build as well as during runtime on the host operating system. This breaks binary portability and also adds unnecessary hurdles on Windows.
   * Furthermore, whilst HID devices are supported by `libusb`, the OS on Macos and Windows explicitly takes over these devices, so only native system calls can be used on recent versions (i.e. you **cannot** use `libusb` for HID).
 * There is a fork of `gousb` [created by @karalabe](https://github.com/karalabe/gousb) that statically linked `libusb` during build, but with the lack of HID access, that work was abandoned.
 * For HID-only devices, a previous self-contained package was created at [`github.com/karalabe/hid`](https://github.com/karalabe/hid), which worked well for hardware wallet uses cases in [`go-ethereum`](https://github.com/ethereum/go-ethereum). It's a simple package that does it's thing well.
   * Unfortunately, `hid` is not capable of talking to generic USB devices. When multiple different devices are needed, eventually some will not support the HID spec (e.g. WebUSB). Pulling in both `hid` and `gousb` will break down due to both depending internally on different versions of `libusb` on Linux.

This `usb` package is a proper integration of `hidapi` and `libusb` so that communication with HID devices is done via system calls, whereas communication with lower level USB devices is done via interrupts. All this detail is hidden away behind a tiny interface.

The package supports Linux, macOS, Windows and FreeBSD. Exclude constraints are also specified for Android and iOS to allow smoother vendoring into cross platform projects.

## Cross-compiling

Using `go get`, the embedded C library is compiled into the binary format of your host OS. Cross compiling to a different platform or architecture entails disabling CGO by default in Go, causing device enumeration `hid.Enumerate()` to yield no results.

To cross compile a functional version of this library, you'll need to enable CGO during cross compilation via `CGO_ENABLED=1` and you'll need to install and set a cross compilation enabled C toolkit via `CC=your-cross-gcc`.

## Acknowledgements

Although the `usb` package is an implementation from scratch, HID support was heavily inspired by the existing [`go.hid`](https://github.com/GeertJohan/go.hid) library, which seems abandoned since 2015; is incompatible with Go 1.6+; and has various external dependencies.

Wide character support in the HID support is done via the [`gowchar`](https://github.com/orofarne/gowchar) library, unmaintained since 2013; non buildable with a modern Go release and failing `go vet` checks. As such, `gowchar` was also vendored in inline.

Error handling for the `libusb` integration originates from the [`gousb`](https://github.com/google/gousb) library.

## License

This USB library is licensed under the [GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html) (dictated by libusb).

If you are only interested in Human Interface devices, a less restrictive package can be found at [`github.com/karalabe/hid`](https://github.com/karalabe/hid).
