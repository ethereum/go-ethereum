package core

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trezor/trezord-go/memorywriter"
	"github.com/trezor/trezord-go/wire"
)

// Package with "core logic" of device listing
// and dealing with sessions, mutexes, ...
//
// USB package is not imported for efficiency
// reasons - USB package uses imports /usb/lowlevel and
// /usb/lowlevel uses cgo, so it takes about 25 seconds to build;
// so building just this package  on its own
// takes less seconds when we dont import USB
// package and use abstract interfaces instead

// USB* interfaces are implemented in usb package

type USBBus interface {
	Enumerate() ([]USBInfo, error)
	Connect(
		path string,
		debug bool, // debug link
		reset bool, // reset is optional, to prevent reseting calls
	) (USBDevice, error)
	Has(path string) bool

	Close() // called on program exit
}

type DeviceType int

const (
	TypeT1Hid        DeviceType = 0
	TypeT1Webusb     DeviceType = 1
	TypeT1WebusbBoot DeviceType = 2
	TypeT2           DeviceType = 3
	TypeT2Boot       DeviceType = 4
	TypeEmulator     DeviceType = 5
)

type USBInfo struct {
	Path      string
	VendorID  int
	ProductID int
	Type      DeviceType
	Debug     bool // has debug enabled?
}

type USBDevice interface {
	io.ReadWriter
	Close(disconnected bool) error
}

type session struct {
	path string
	id   string
	dev  USBDevice
	call int32 // atomic
}

type EnumerateEntry struct {
	Path    string     `json:"path"`
	Vendor  int        `json:"vendor"`
	Product int        `json:"product"`
	Type    DeviceType `json:"-"`     // used only in status page, not in JSON
	Debug   bool       `json:"debug"` // has debug enabled?

	Session      *string `json:"session"`
	DebugSession *string `json:"debugSession"`
}

type EnumerateEntries []EnumerateEntry

func (entries EnumerateEntries) Len() int {
	return len(entries)
}
func (entries EnumerateEntries) Less(i, j int) bool {
	return entries[i].Path < entries[j].Path
}
func (entries EnumerateEntries) Swap(i, j int) {
	entries[i], entries[j] = entries[j], entries[i]
}

type Core struct {
	bus USBBus

	normalSessions map[string]*session
	debugSessions  map[string]*session
	sessionsMutex  sync.Mutex // for atomic access to sessions

	allowStealing bool
	reset         bool

	// We cannot make calls and enumeration at the same time,
	// because of some libusb/hidapi issues.
	// However, it is easier to fix it here than in the usb/ packages,
	// because they don't know about whole messages and just send
	// small packets by read/write
	//
	// Those variables help with that
	callsInProgress int
	callMutex       sync.Mutex
	lastInfosMutex  sync.Mutex
	lastInfos       []USBInfo // when call is in progress, use saved info for enumerating
	// note - both lastInfos and Enumerate result have "paths"
	// as *fake paths* 2.0.26 onwards; just using device IDs,
	// unique for the device

	// the paths we present out are not actual paths
	// from 2.0.26 onwards;
	// it's just an ID of a device
	// we keep the IDs here
	usbPaths  map[int]string // id => path
	biggestID int

	log *memorywriter.MemoryWriter

	latestSessionID int
}

var (
	ErrWrongPrevSession = errors.New("wrong previous session")
	ErrSessionNotFound  = errors.New("session not found")
	ErrMalformedData    = errors.New("malformed data")
	ErrOtherCall        = errors.New("other call in progress")
)

const (
	VendorT1            = 0x534c
	ProductT1Firmware   = 0x0001
	VendorT2            = 0x1209
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
)

func New(bus USBBus, log *memorywriter.MemoryWriter, allowStealing, reset bool) *Core {
	c := &Core{
		bus:            bus,
		normalSessions: make(map[string]*session),
		debugSessions:  make(map[string]*session),
		log:            log,
		allowStealing:  allowStealing,
		reset:          reset,
		usbPaths:       make(map[int]string),
	}
	go c.backgroundListen()
	return c
}

const (
	iterMax   = 600
	iterDelay = 500 // ms
)

// This is here just to force recomputing the IDs of
// the disconnected devices (c.usbPaths)
// Note - this does not do anything when no device is connected
// or when no enumerate is run first...
// -> it runs whenever someone calls Enumerate/Listen
// and there are some devices left
// It does not spam USB that much more than listen itself
func (c *Core) backgroundListen() {
	for {
		time.Sleep(iterDelay * time.Millisecond)

		c.lastInfosMutex.Lock()
		linfos := len(c.lastInfos)
		c.lastInfosMutex.Unlock()
		if linfos > 0 {
			c.log.Log("background enum runs")
			_, err := c.Enumerate()
			if err != nil {
				// we dont really care here
				c.log.Log("error - " + err.Error())
			}
		}
	}
}

func (c *Core) saveUsbPaths(devs []USBInfo) (res []USBInfo) {
	for _, dev := range devs {
		add := true
		for _, usbPath := range c.usbPaths {
			if dev.Path == usbPath {
				add = false
			}
		}
		if add {
			newID := c.biggestID + 1
			c.biggestID = newID
			c.usbPaths[newID] = dev.Path
		}
	}
	reverse := make(map[string]string)
	for id, usbPath := range c.usbPaths {
		discard := true
		for _, dev := range devs {
			if dev.Path == usbPath {
				discard = false
			}
		}
		if discard {
			delete(c.usbPaths, id)
		} else {
			reverse[usbPath] = strconv.Itoa(id)
		}
	}
	for _, dev := range devs {
		res = append(res, USBInfo{
			Path:      reverse[dev.Path],
			VendorID:  dev.VendorID,
			ProductID: dev.ProductID,
			Type:      dev.Type,
			Debug:     dev.Debug,
		})
	}
	return
}

func (c *Core) Enumerate() ([]EnumerateEntry, error) {
	// Lock for atomic access to s.sessions.
	c.log.Log("locking sessionsMutex")
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	c.log.Log("locking callMutex")
	// Lock for atomic access to s.callInProgress.  It needs to be over
	// whole function, so that call does not actually start while
	// enumerating.
	c.callMutex.Lock()
	defer c.callMutex.Unlock()

	c.lastInfosMutex.Lock()
	defer c.lastInfosMutex.Unlock()

	// Use saved info if call is in progress, otherwise enumerate.
	infos := c.lastInfos

	c.log.Log(fmt.Sprintf("callsInProgress %d", c.callsInProgress))
	if c.callsInProgress == 0 {
		c.log.Log("bus")
		busInfos, err := c.bus.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = c.saveUsbPaths(busInfos)
		c.lastInfos = infos
	}

	entries := c.createEnumerateEntries(infos)
	c.log.Log("release disconnected")
	c.releaseDisconnected(infos, false)
	c.releaseDisconnected(infos, true)
	return entries, nil
}

func (c *Core) findSession(e *EnumerateEntry, path string, debug bool) {
	for _, ss := range c.sessions(debug) {
		if ss.path == path {
			// Copying to prevent overwriting on Acquire and
			// wrong comparison in Listen.
			ssidCopy := ss.id
			if debug {
				e.DebugSession = &ssidCopy
			} else {
				e.Session = &ssidCopy
			}
		}
	}
}

func (c *Core) createEnumerateEntry(info USBInfo) EnumerateEntry {
	e := EnumerateEntry{
		Path:    info.Path,
		Vendor:  info.VendorID,
		Product: info.ProductID,
		Type:    info.Type,
		Debug:   info.Debug,
	}
	c.findSession(&e, info.Path, false)
	c.findSession(&e, info.Path, true)
	return e
}

func (c *Core) createEnumerateEntries(infos []USBInfo) EnumerateEntries {
	entries := make(EnumerateEntries, 0, len(infos))
	for _, info := range infos {
		e := c.createEnumerateEntry(info)
		entries = append(entries, e)
	}
	entries.Sort()
	return entries
}

func (entries EnumerateEntries) Sort() {
	sort.Sort(entries)
}

func (c *Core) sessions(debug bool) map[string]*session {
	sessions := c.normalSessions
	if debug {
		sessions = c.debugSessions
	}
	return sessions
}

func (c *Core) releaseDisconnected(infos []USBInfo, debug bool) {

	for ssid, ss := range c.sessions(debug) {
		connected := false
		for _, info := range infos {
			if ss.path == info.Path {
				connected = true
			}
		}
		if !connected {
			c.log.Log(fmt.Sprintf("disconnected device %s", ssid))
			err := c.release(ssid, true, debug)
			// just log if there is an error
			// they are disconnected anyway
			if err != nil {
				c.log.Log(fmt.Sprintf("Error on releasing disconnected device: %s", err))
			}
		}
	}
}

func (c *Core) Release(session string, debug bool) error {
	return c.release(session, false, debug)
}

func (c *Core) release(
	session string,
	disconnected bool,
	debug bool,
) error {
	c.log.Log(fmt.Sprintf("session %s", session))
	acquired := (c.sessions(debug))[session]
	if acquired == nil {
		c.log.Log("session not found")
		return ErrSessionNotFound
	}
	delete(c.sessions(debug), session)

	c.log.Log("bus close")
	err := acquired.dev.Close(disconnected)
	return err
}

func (c *Core) Listen(entries []EnumerateEntry, ctx context.Context) ([]EnumerateEntry, error) {
	c.log.Log("start")

	EnumerateEntries(entries).Sort()

	for i := 0; i < iterMax; i++ {
		c.log.Log("before enumerating")
		e, enumErr := c.Enumerate()
		if enumErr != nil {
			return nil, enumErr
		}
		for i := range e {
			e[i].Type = 0 // type is not exported/imported to json
		}
		if reflect.DeepEqual(entries, e) {
			c.log.Log("equal, waiting")
			select {
			case <-ctx.Done():
				c.log.Log(fmt.Sprintf("request closed (%s)", ctx.Err().Error()))
				return nil, nil
			default:
				time.Sleep(iterDelay * time.Millisecond)
			}
		} else {
			c.log.Log("different")
			entries = e
			break
		}
	}
	c.log.Log("encoding and exiting")
	return entries, nil
}

func (c *Core) findPrevSession(path string, debug bool) string {
	// note - sessionsMutex must be locked before entering this
	for _, ss := range c.sessions(debug) {
		if ss.path == path {
			return ss.id
		}
	}

	return ""
}

func (c *Core) Acquire(
	path, prev string,
	debug bool,
) (string, error) {
	// note - path is *fake path*, basically device ID,
	// because that is what enumerate returns;
	// we convert it to actual path for USB layer

	c.log.Log("locking sessionsMutex")
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()

	c.log.Log(fmt.Sprintf("input path %s prev %s", path, prev))

	prevSession := c.findPrevSession(path, debug)

	c.log.Log(fmt.Sprintf("actually previous %s", prevSession))

	if prevSession != prev {
		return "", ErrWrongPrevSession
	}

	if (!c.allowStealing) && prevSession != "" {
		return "", ErrOtherCall
	}

	if prev != "" {
		c.log.Log("releasing previous")
		err := c.release(prev, false, debug)
		if err != nil {
			return "", err
		}
	}

	// reset device ONLY if no call on the other port
	// otherwise, USB reset stops other call
	otherSession := c.findPrevSession(path, !debug)
	reset := otherSession == "" && c.reset

	pathI, err := strconv.Atoi(path)
	if err != nil {
		return "", err
	}

	usbPath, exists := c.usbPaths[pathI]
	if !exists {
		return "", errors.New("device not found")
	}

	c.log.Log("trying to connect")
	dev, err := c.tryConnect(usbPath, debug, reset)
	if err != nil {
		return "", err
	}

	id := c.newSession(debug)

	sess := &session{
		path: path,
		dev:  dev,
		call: 0,
		id:   id,
	}

	c.log.Log(fmt.Sprintf("new session is %s", id))

	(c.sessions(debug))[id] = sess

	return id, nil
}

// Chrome tries to read from trezor immediately after connecting,
// ans so do we.  Bad timing can produce error on s.bus.Connect.
// Try 3 times with a 100ms delay.
func (c *Core) tryConnect(path string, debug bool, reset bool) (USBDevice, error) {
	tries := 0
	for {
		c.log.Log(fmt.Sprintf("try number %d", tries))
		dev, err := c.bus.Connect(path, debug, reset)
		if err != nil {
			if tries < 3 {
				c.log.Log("sleeping")
				tries++
				time.Sleep(100 * time.Millisecond)
			} else {
				c.log.Log("tryConnect - too many times, exiting")
				return nil, err
			}
		} else {
			return dev, nil
		}
	}
}

func (c *Core) newSession(debug bool) string {
	c.latestSessionID++
	res := strconv.Itoa(c.latestSessionID)
	if debug {
		res = "debug" + res
	}
	return res
}

type CallMode int

const (
	CallModeRead      CallMode = 0
	CallModeWrite     CallMode = 1
	CallModeReadWrite CallMode = 2
)

func (c *Core) Call(
	body []byte,
	session string,
	mode CallMode,
	debug bool,
	ctx context.Context,
) ([]byte, error) {
	c.log.Log("callMutex lock")
	c.callMutex.Lock()

	c.log.Log("callMutex set callInProgress true, unlock")
	c.callsInProgress++

	c.callMutex.Unlock()
	c.log.Log("callMutex unlock done")

	defer func() {
		c.log.Log("callMutex closing lock")
		c.callMutex.Lock()

		c.log.Log("callMutex set callInProgress false, unlock")
		c.callsInProgress--

		c.callMutex.Unlock()
		c.log.Log("callMutex closing unlock")
	}()

	c.log.Log("sessionsMutex lock")
	c.sessionsMutex.Lock()
	acquired := (c.sessions(debug))[session]
	c.sessionsMutex.Unlock()
	c.log.Log("sessionsMutex unlock done")

	if acquired == nil {
		return nil, ErrSessionNotFound
	}

	c.log.Log("checking other call on same session")
	freeToCall := atomic.CompareAndSwapInt32(&acquired.call, 0, 1)
	if !freeToCall {
		return nil, ErrOtherCall
	}

	c.log.Log("checking other call on same session done")
	defer func() {
		atomic.StoreInt32(&acquired.call, 0)
	}()

	finished := make(chan bool, 1)
	defer func() {
		finished <- true
	}()

	go func() {
		select {
		case <-finished:
			return
		case <-ctx.Done():
			c.log.Log(fmt.Sprintf("detected request close %s, auto-release", ctx.Err().Error()))
			errRelease := c.release(session, false, debug)
			if errRelease != nil {
				// just log, since request is already closed
				c.log.Log(fmt.Sprintf("Error while releasing: %s", errRelease.Error()))
			}
		}
	}()

	c.log.Log("before actual logic")
	bytes, err := c.readWriteDev(body, acquired.dev, mode)
	c.log.Log("after actual logic")

	return bytes, err
}

func (c *Core) writeDev(body []byte, device io.Writer) error {
	c.log.Log("decodeRaw")
	msg, err := c.decodeRaw(body)
	if err != nil {
		return err
	}

	c.log.Log("writeTo")
	_, err = msg.WriteTo(device)
	return err
}

func (c *Core) readDev(device io.Reader) ([]byte, error) {
	c.log.Log("readFrom")
	msg, err := wire.ReadFrom(device, c.log)
	if err != nil {
		return nil, err
	}

	c.log.Log("encoding back")
	return c.encodeRaw(msg)
}

func (c *Core) readWriteDev(
	body []byte,
	device io.ReadWriter,
	mode CallMode,
) ([]byte, error) {

	if mode == CallModeRead {
		if len(body) != 0 {
			return nil, errors.New("non-empty body on read mode")
		}
		c.log.Log("skipping write")
	} else {
		err := c.writeDev(body, device)
		if err != nil {
			return nil, err
		}
	}

	if mode == CallModeWrite {
		c.log.Log("skipping read")
		return []byte{0}, nil
	}
	return c.readDev(device)
}

func (c *Core) decodeRaw(body []byte) (*wire.Message, error) {
	c.log.Log("readAll")

	c.log.Log("decodeString")

	if len(body) < 6 {
		c.log.Log("body too short")
		return nil, ErrMalformedData
	}

	kind := binary.BigEndian.Uint16(body[0:2])
	size := binary.BigEndian.Uint32(body[2:6])
	data := body[6:]
	if uint32(len(data)) != size {
		c.log.Log("wrong data length")
		return nil, ErrMalformedData
	}

	if wire.Validate(data) != nil {
		c.log.Log("invalid data")
		return nil, ErrMalformedData
	}

	c.log.Log("returning")
	return &wire.Message{
		Kind: kind,
		Data: data,

		Log: c.log,
	}, nil
}

func (c *Core) encodeRaw(msg *wire.Message) ([]byte, error) {
	c.log.Log("start")
	var header [6]byte
	data := msg.Data
	kind := msg.Kind
	size := uint32(len(msg.Data))

	binary.BigEndian.PutUint16(header[0:2], kind)
	binary.BigEndian.PutUint32(header[2:6], size)

	res := append(header[:], data...)

	return res, nil
}
