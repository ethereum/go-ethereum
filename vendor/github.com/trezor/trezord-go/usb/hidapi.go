// +build darwin,!ios,cgo windows,cgo

package usb

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	lowlevel "github.com/trezor/trezord-go/usb/lowlevel/hidapi"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
)

const (
	hidapiPrefix = "hid"
	hidUsagePage = 0xFF00
	hidTimeout   = 50
	HIDUse       = true
)

type HIDAPI struct {
	mw *memorywriter.MemoryWriter
}

func InitHIDAPI(mw *memorywriter.MemoryWriter) (*HIDAPI, error) {
	lowlevel.SetLogWriter(mw)
	return &HIDAPI{
		mw: mw,
	}, nil
}

func (b *HIDAPI) Enumerate() ([]core.USBInfo, error) {
	var infos []core.USBInfo

	b.mw.Log("low level")
	devs := lowlevel.HidEnumerate(0, 0)

	b.mw.Log("low level done")

	for _, dev := range devs { // enumerate all devices
		if b.match(&dev) {
			infos = append(infos, core.USBInfo{
				Path:      b.identify(&dev),
				VendorID:  int(dev.VendorID),
				ProductID: int(dev.ProductID),
				Type:      core.TypeT1Hid,
				Debug:     false,
			})
		}
	}
	return infos, nil
}

func (b *HIDAPI) Has(path string) bool {
	return strings.HasPrefix(path, hidapiPrefix)
}

func (b *HIDAPI) Connect(path string, debug bool, reset bool) (core.USBDevice, error) {
	if debug {
		return nil, errNotDebug
	}
	b.mw.Log("enumerate to find")
	devs := lowlevel.HidEnumerate(0, 0)
	b.mw.Log("enumerate done")

	for _, dev := range devs { // enumerate all devices
		if b.match(&dev) && b.identify(&dev) == path {
			b.mw.Log("low level open")
			d, err := dev.Open()
			if err != nil {
				return nil, err
			}
			b.mw.Log("detecting prepend")
			prepend, err := b.detectPrepend(d)
			if err != nil {
				return nil, err
			}
			b.mw.Log(fmt.Sprintf("done (prepend %t)", prepend))
			return &HID{
				dev:     d,
				prepend: prepend,
				mw:      b.mw,
			}, nil
		}
	}
	return nil, ErrNotFound
}

func (b *HIDAPI) match(d *lowlevel.HidDeviceInfo) bool {
	vid := d.VendorID
	pid := d.ProductID
	trezor1 := vid == core.VendorT1 && (pid == core.ProductT1Firmware)
	trezor2 := vid == core.VendorT2 && (pid == core.ProductT2Firmware || pid == core.ProductT2Bootloader)
	return (trezor1 || trezor2) && (d.Interface == int(normalIface.number) || d.UsagePage == hidUsagePage)
}

func (b *HIDAPI) identify(dev *lowlevel.HidDeviceInfo) string {
	path := []byte(dev.Path)
	digest := sha256.Sum256(path)
	return hidapiPrefix + hex.EncodeToString(digest[:])
}

func (b *HIDAPI) Close() {
	// nothing
}

type HID struct {
	dev     *lowlevel.HidDevice
	prepend bool // on windows, see detectPrepend

	closed        int32 // atomic
	transferMutex sync.Mutex
	// closing cannot happen while read/write is hapenning,
	// otherwise it segfaults on windows

	mw *memorywriter.MemoryWriter
}

func (d *HID) Close(disconnected bool) error {

	d.mw.Log("storing d.closed")
	atomic.StoreInt32(&d.closed, 1)

	d.mw.Log("wait for transferMutex lock")
	d.transferMutex.Lock()
	d.mw.Log("low level close")
	err := d.dev.Close()
	d.transferMutex.Unlock()

	d.mw.Log("done")

	return err
}

var unknownErrorMessage = "hidapi: unknown failure"

// This will write a useless buffer to trezor
// to test whether it is an older HID version on reportid 63
// or a newer one that is on id 0.
// The older one does not need prepending, the newer one does
// This makes difference only on windows
func (b *HIDAPI) detectPrepend(dev *lowlevel.HidDevice) (bool, error) {
	buf := []byte{63}
	for i := 0; i < 63; i++ {
		buf = append(buf, 0xff)
	}

	// first test newer version
	w, err := dev.Write(buf, true)
	if w == 65 {
		return true, nil
	}
	if err != nil {
		b.mw.Log("found older version - error")
		b.mw.Log(err.Error())
	}

	// then test older version
	w, err = dev.Write(buf, false)
	if err != nil {
		return false, err
	}
	if w == 64 {
		return false, nil
	}

	return false, errors.New("unknown HID version")
}

func (d *HID) readWrite(buf []byte, read bool) (int, error) {

	d.mw.Log("start")
	for {
		d.mw.Log("checking closed")
		closed := (atomic.LoadInt32(&d.closed)) == 1
		if closed {
			d.mw.Log("closed, skip")
			return 0, errClosedDevice
		}

		d.mw.Log("lock transfer mutex")
		d.transferMutex.Lock()
		d.mw.Log("actual interrupt transport")

		var w int
		var err error

		if read {
			d.mw.Log("r start")
			w, err = d.dev.Read(buf, hidTimeout)
			d.mw.Log("r end")
		} else {
			d.mw.Log("w start")
			w, err = d.dev.Write(buf, d.prepend)
			d.mw.Log("w end")
		}

		d.transferMutex.Unlock()
		d.mw.Log("single transfer done")

		if err == nil {
			// sometimes, empty report is read, skip it
			if w > 0 {
				d.mw.Log("single transfer succesful")
				return w, err
			}
			if !read {
				return 0, errors.New("HID - empty write")
			}

			d.mw.Log("skipping empty transfer - go again")
		} else {
			if err.Error() == unknownErrorMessage {
				return 0, errDisconnect
			}
			return 0, err
		}

		// continue the for cycle
	}
}

func (d *HID) Write(buf []byte) (int, error) {
	return d.readWrite(buf, false)
}

func (d *HID) Read(buf []byte) (int, error) {
	return d.readWrite(buf, true)
}
