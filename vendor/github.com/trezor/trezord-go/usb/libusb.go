package usb

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	lowlevel "github.com/trezor/trezord-go/usb/lowlevel/libusb"

	"github.com/trezor/trezord-go/core"
	"github.com/trezor/trezord-go/memorywriter"
)

const (
	libusbPrefix   = "lib"
	usbConfigNum   = 1
	usbConfigIndex = 0
)

type libusbIfaceData struct {
	number     uint8
	altSetting uint8
	epIn       uint8
	epOut      uint8
}

var normalIface = libusbIfaceData{
	number:     0,
	altSetting: 0,
	epIn:       0x81,
	epOut:      0x01,
}

// Old bootloader has different epOut
// We need it here, since on Linux,
// we use libusb instead of hidapi for old BL
var oldBLIface = libusbIfaceData{
	number:     0,
	altSetting: 0,
	epIn:       0x81,
	epOut:      0x02,
}

var debugIface = libusbIfaceData{
	number:     1,
	altSetting: 0,
	epIn:       0x82,
	epOut:      0x02,
}

type LibUSB struct {
	usb    lowlevel.Context
	mw     *memorywriter.MemoryWriter
	only   bool
	cancel bool
	detach bool
}

func InitLibUSB(mw *memorywriter.MemoryWriter, onlyLibusb, allowCancel, detach bool) (*LibUSB, error) {
	var usb lowlevel.Context
	mw.Log("init")
	lowlevel.SetLogWriter(mw)

	err := lowlevel.Init(&usb)
	if err != nil {
		return nil, fmt.Errorf(`error when initializing LibUSB.
If you run trezord in an environment without USB (for example, docker or travis), use '-u=false'. For example, './trezord-go -e 21324 -u=false'.

Original error: %v`, err)
	}

	mw.Log("init done")

	return &LibUSB{
		usb:    usb,
		mw:     mw,
		only:   onlyLibusb,
		cancel: allowCancel,
		detach: detach,
	}, nil
}

func (b *LibUSB) Close() {
	b.mw.Log("all close (should happen only on exit)")
	lowlevel.Exit(b.usb)
}

func hasIface(dev lowlevel.Device, dIface libusbIfaceData, dClass uint8) (bool, error) {
	config, err := lowlevel.Get_Config_Descriptor(dev, usbConfigIndex)
	if err != nil {
		return false, err
	}

	ifaces := config.Interface
	for _, iface := range ifaces {
		for _, alt := range iface.Altsetting {
			if alt.BInterfaceNumber == dIface.number &&
				alt.BAlternateSetting == dIface.altSetting &&
				alt.BNumEndpoints == 2 &&
				alt.BInterfaceClass == dClass &&
				alt.Endpoint[0].BEndpointAddress == dIface.epIn &&
				alt.Endpoint[1].BEndpointAddress == dIface.epOut {
				return true, nil
			}
		}
	}
	return false, nil
}

func detectDebug(dev lowlevel.Device) (bool, error) {
	return hasIface(dev, debugIface, uint8(lowlevel.CLASS_VENDOR_SPEC))
}

func detectOldBL(dev lowlevel.Device) (bool, error) {
	return hasIface(dev, oldBLIface, uint8(lowlevel.CLASS_HID))
}

func (b *LibUSB) Enumerate() ([]core.USBInfo, error) {
	b.mw.Log("low level enumerating")
	list, err := lowlevel.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.mw.Log("low level enumerating done")

	defer func() {
		b.mw.Log("freeing device list")
		lowlevel.Free_Device_List(list, 1) // unlink devices
		b.mw.Log("freeing device list done")
	}()

	var infos []core.USBInfo

	// There is a bug in libusb that makes
	// device appear twice with the same path.
	// This is already fixed in libusb 2.0.12;
	// however, 2.0.12 has other problems with windows, so we
	// patchfix it here
	paths := make(map[string]bool)

	for _, dev := range list {
		m, t := b.match(dev)
		if m {
			b.mw.Log("getting device descriptor")
			dd, err := lowlevel.Get_Device_Descriptor(dev)
			if err != nil {
				b.mw.Log("error getting device descriptor " + err.Error())
				continue
			}
			path := b.identify(dev)
			inset := paths[path]
			if !inset {
				debug, err := detectDebug(dev)
				if err != nil {
					b.mw.Log("error detecting debug " + err.Error())
					continue
				}
				infos = append(infos, core.USBInfo{
					Path:      path,
					VendorID:  int(dd.IdVendor),
					ProductID: int(dd.IdProduct),
					Type:      t,
					Debug:     debug,
				})
				paths[path] = true
			}
		}
	}
	return infos, nil
}

func (b *LibUSB) Has(path string) bool {
	return strings.HasPrefix(path, libusbPrefix)
}

func (b *LibUSB) Connect(path string, debug bool, reset bool) (core.USBDevice, error) {
	b.mw.Log("low level enumerating")
	list, err := lowlevel.Get_Device_List(b.usb)

	if err != nil {
		return nil, err
	}
	b.mw.Log("low level enumerating done")

	defer func() {
		b.mw.Log("freeing device list")
		lowlevel.Free_Device_List(list, 1) // unlink devices
		b.mw.Log("freeing device list done")
	}()

	// There is a bug in libusb that makes
	// device appear twice with the same path.
	// This is already fixed in libusb 2.0.12;
	// however, 2.0.12 has other problems with windows, so we
	// patchfix it here
	mydevs := make([]lowlevel.Device, 0)
	for _, dev := range list {
		m, _ := b.match(dev)
		if m && b.identify(dev) == path {
			mydevs = append(mydevs, dev)
		}
	}

	err = ErrNotFound
	for _, dev := range mydevs {
		res, errConn := b.connect(dev, debug, reset)
		if errConn == nil {
			return res, nil
		}
		err = errConn
	}
	return nil, err
}

func (b *LibUSB) setConfiguration(d lowlevel.Device_Handle) {
	currConf, err := lowlevel.Get_Configuration(d)
	if err != nil {
		b.mw.Log(fmt.Sprintf("current configuration err %s", err.Error()))
	} else {
		b.mw.Log(fmt.Sprintf("current configuration %d", currConf))
	}
	if currConf == usbConfigNum {
		b.mw.Log("not setting config, same")
	} else {
		b.mw.Log("set_configuration")
		err = lowlevel.Set_Configuration(d, usbConfigNum)
		if err != nil {
			// don't abort if set configuration fails
			// lowlevel.Close(d)
			// return nil, err
			b.mw.Log(fmt.Sprintf("Warning: error at configuration set: %s", err))
		}

		currConf, err = lowlevel.Get_Configuration(d)
		if err != nil {
			b.mw.Log(fmt.Sprintf("current configuration err %s", err.Error()))
		} else {
			b.mw.Log(fmt.Sprintf("current configuration %d", currConf))
		}
	}
}

func (b *LibUSB) claimInterface(d lowlevel.Device_Handle, debug bool) (bool, error) {
	attach := false
	usbIfaceNum := int(normalIface.number)
	if debug {
		usbIfaceNum = int(debugIface.number)
	}
	if b.detach {
		b.mw.Log("detecting kernel driver")
		kernel, errD := lowlevel.Kernel_Driver_Active(d, usbIfaceNum)
		if errD != nil {
			b.mw.Log("detecting kernel driver failed")
			lowlevel.Close(d)
			return false, errD
		}
		if kernel {
			attach = true
			b.mw.Log("kernel driver active, detach")
			errD = lowlevel.Detach_Kernel_Driver(d, usbIfaceNum)
			if errD != nil {
				b.mw.Log("detaching kernel driver failed")
				lowlevel.Close(d)
				return false, errD
			}
		}
	}
	b.mw.Log("claiming interface")
	err := lowlevel.Claim_Interface(d, usbIfaceNum)
	if err != nil {
		b.mw.Log("claiming interface failed")
		lowlevel.Close(d)
		return false, err
	}

	b.mw.Log("claiming interface done")

	return attach, nil
}

func (b *LibUSB) connect(dev lowlevel.Device, debug bool, reset bool) (*LibUSBDevice, error) {

	b.mw.Log("detect old BL")
	oldBL, err := detectOldBL(dev)
	if err != nil {
		return nil, err
	}

	b.mw.Log("low level")
	d, err := lowlevel.Open(dev)
	if err != nil {
		return nil, err
	}
	b.mw.Log("reset")
	if reset {
		err = lowlevel.Reset_Device(d)
		if err != nil {
			// don't abort if reset fails
			// lowlevel.Close(d)
			// return nil, err
			b.mw.Log(fmt.Sprintf("Warning: error at device reset: %s", err))
		}
	}

	b.setConfiguration(d)
	attach, err := b.claimInterface(d, debug)
	if err != nil {
		return nil, err
	}
	return &LibUSBDevice{
		dev:    d,
		closed: 0,

		mw:     b.mw,
		cancel: b.cancel,
		attach: attach,
		debug:  debug,
		oldBL:  oldBL,
	}, nil
}

func matchType(dd *lowlevel.Device_Descriptor) core.DeviceType {
	if dd.IdProduct == core.ProductT1Firmware {
		// this is HID, in platforms where we don't use hidapi (linux, bsd)
		return core.TypeT1Hid
	}

	if dd.IdProduct == core.ProductT2Bootloader {
		if int(dd.BcdDevice>>8) == 1 {
			return core.TypeT1WebusbBoot
		}
		return core.TypeT2Boot
	}

	if int(dd.BcdDevice>>8) == 1 {
		return core.TypeT1Webusb
	}

	return core.TypeT2
}

func (b *LibUSB) match(dev lowlevel.Device) (bool, core.DeviceType) {
	b.mw.Log("start")
	dd, err := lowlevel.Get_Device_Descriptor(dev)
	if err != nil {
		b.mw.Log("error getting descriptor -" + err.Error())
		return false, 0
	}

	vid := dd.IdVendor
	pid := dd.IdProduct
	if !b.matchVidPid(vid, pid) {
		b.mw.Log("unmatched")
		return false, 0
	}

	b.mw.Log("matched, get active config")
	c, err := lowlevel.Get_Active_Config_Descriptor(dev)
	if err != nil {
		b.mw.Log("error getting config descriptor " + err.Error())
		return false, 0
	}

	b.mw.Log("let's test")

	var is bool
	usbIfaceNum := normalIface.number
	usbAltSetting := normalIface.altSetting
	if b.only {

		// if we don't use hidapi at all, keep HID devices
		is = (c.BNumInterfaces > usbIfaceNum &&
			c.Interface[usbIfaceNum].Num_altsetting > int(usbAltSetting))

	} else {

		is = (c.BNumInterfaces > usbIfaceNum &&
			c.Interface[usbIfaceNum].Num_altsetting > int(usbAltSetting) &&
			c.Interface[usbIfaceNum].Altsetting[usbAltSetting].BInterfaceClass == lowlevel.CLASS_VENDOR_SPEC)
	}

	if !is {
		b.mw.Log("not matched")
		return false, 0
	}
	b.mw.Log("matched")
	return true, matchType(dd)

}

func (b *LibUSB) matchVidPid(vid uint16, pid uint16) bool {
	// Note: Trezor1 libusb will actually have the T2 vid/pid
	trezor2 := vid == core.VendorT2 && (pid == core.ProductT2Firmware || pid == core.ProductT2Bootloader)

	if b.only {
		trezor1 := vid == core.VendorT1 && (pid == core.ProductT1Firmware)
		return trezor1 || trezor2
	}

	return trezor2
}

func (b *LibUSB) identify(dev lowlevel.Device) string {
	var ports [8]byte
	p, err := lowlevel.Get_Port_Numbers(dev, ports[:])
	if err != nil {
		b.mw.Log(fmt.Sprintf("error getting port numbers %s", err.Error()))
		return ""
	}
	return libusbPrefix + hex.EncodeToString(p)
}

type LibUSBDevice struct {
	dev lowlevel.Device_Handle

	closed              int32 // atomic
	normalTransferMutex sync.Mutex
	debugTransferMutex  sync.Mutex
	// two interrupt_transfers should not happen at the same time

	cancel bool
	attach bool
	debug  bool

	oldBL bool

	mw *memorywriter.MemoryWriter
}

func (d *LibUSBDevice) Close(disconnected bool) error {
	d.mw.Log("storing d.closed")
	atomic.StoreInt32(&d.closed, 1)

	if d.cancel {
		// libusb close does NOT cancel transfers on close
		// => we are using our own function that we added to libusb/sync.c
		// this "unblocks" Interrupt_Transfer in readWrite

		d.mw.Log("canceling previous transfers")
		lowlevel.Cancel_Sync_Transfers_On_Device(d.dev)

		// reading recently disconnected device sometimes causes weird issues
		// => if we *know* it is disconnected, don't finish read queue
		//
		// Finishing read queue is not necessary when we don't allow cancelling
		// (since when we don't allow cancelling, we don't allow session stealing)
		if !disconnected {
			d.mw.Log("finishing read queue")
			d.finishReadQueue(d.debug)
		}
	}

	d.mw.Log("releasing interface")
	iface := int(normalIface.number)
	if d.debug {
		iface = int(debugIface.number)
	}
	err := lowlevel.Release_Interface(d.dev, iface)
	if err != nil {
		// do not throw error, it is just release anyway
		d.mw.Log(fmt.Sprintf("Warning: error at releasing interface: %s", err))
	}

	if d.attach {
		err = lowlevel.Attach_Kernel_Driver(d.dev, iface)
		if err != nil {
			// do not throw error, it is just re-attach anyway
			d.mw.Log(fmt.Sprintf("Warning: error at re-attaching driver: %s", err))
		}
	}

	d.mw.Log("low level close")
	lowlevel.Close(d.dev)
	d.mw.Log("done")

	return nil
}

func (d *LibUSBDevice) transferMutexLock(debug bool) {
	if debug {
		d.debugTransferMutex.Lock()
	} else {
		d.normalTransferMutex.Lock()
	}
}

func (d *LibUSBDevice) transferMutexUnlock(debug bool) {
	if debug {
		d.debugTransferMutex.Unlock()
	} else {
		d.normalTransferMutex.Unlock()
	}
}

func (d *LibUSBDevice) finishReadQueue(debug bool) {
	d.mw.Log("wait for transfermutex lock")
	usbEpIn := normalIface.epIn
	if debug {
		usbEpIn = debugIface.epIn
	}
	d.transferMutexLock(debug)
	var err error
	var buf [64]byte

	for err == nil {
		// these transfers have timeouts => should not interfer with
		// cancel_sync_transfers_on_device
		d.mw.Log("transfer")
		_, err = lowlevel.Interrupt_Transfer(d.dev, usbEpIn, buf[:], 50)
	}
	d.transferMutexUnlock(debug)
	d.mw.Log("done")
}

func (d *LibUSBDevice) readWrite(buf []byte, endpoint uint8) (int, error) {
	d.mw.Log("start")
	for {
		d.mw.Log("checking closed")
		closed := (atomic.LoadInt32(&d.closed)) == 1
		if closed {
			d.mw.Log("closed, skip")
			return 0, errClosedDevice
		}

		d.mw.Log("lock transfer mutex")
		d.transferMutexLock(d.debug)
		d.mw.Log("actual interrupt transport")
		// This has no timeout, but is stopped by Cancel_Sync_Transfers_On_Device
		p, err := lowlevel.Interrupt_Transfer(d.dev, endpoint, buf, 0)
		d.transferMutexUnlock(d.debug)
		d.mw.Log("single transfer done")

		if err != nil {
			d.mw.Log(fmt.Sprintf("error seen - %s", err.Error()))
			if isErrorDisconnect(err) {
				d.mw.Log("device probably disconnected")
				return 0, errDisconnect
			}

			d.mw.Log("other error")
			return 0, err
		}

		// sometimes, empty report is read, skip it
		// TODO: is this still needed with 0 timeouts?
		if len(p) > 0 {
			d.mw.Log("single transfer succesful")
			return len(p), err
		}
		d.mw.Log("skipping empty transfer, go again")
		// continue the for cycle if empty transfer
	}
}

func isErrorDisconnect(err error) bool {
	// according to libusb docs, disconnecting device should cause only
	// LIBUSB_ERROR_NO_DEVICE error, but in real life, it causes also
	// LIBUSB_ERROR_IO, LIBUSB_ERROR_PIPE, LIBUSB_ERROR_OTHER

	return (err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_IO)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_NO_DEVICE)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_OTHER)) ||
		err.Error() == lowlevel.Error_Name(int(lowlevel.ERROR_PIPE)))
}

func (d *LibUSBDevice) Write(buf []byte) (int, error) {
	d.mw.Log("write start")
	usbEpOut := normalIface.epOut
	if d.oldBL {
		usbEpOut = oldBLIface.epOut
	}
	if d.debug {
		usbEpOut = debugIface.epOut
	}
	return d.readWrite(buf, usbEpOut)
}

func (d *LibUSBDevice) Read(buf []byte) (int, error) {
	d.mw.Log("read start")
	usbEpIn := normalIface.epIn
	if d.debug {
		usbEpIn = debugIface.epIn
	}
	return d.readWrite(buf, usbEpIn)
}
