package scard

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modwinscard = syscall.NewLazyDLL("winscard.dll")

	procEstablishContext = modwinscard.NewProc("SCardEstablishContext")
	procReleaseContext   = modwinscard.NewProc("SCardReleaseContext")
	procIsValidContext   = modwinscard.NewProc("SCardIsValidContext")
	procCancel           = modwinscard.NewProc("SCardCancel")
	procListReaders      = modwinscard.NewProc("SCardListReadersW")
	procListReaderGroups = modwinscard.NewProc("SCardListReaderGroupsW")
	procGetStatusChange  = modwinscard.NewProc("SCardGetStatusChangeW")
	procConnect          = modwinscard.NewProc("SCardConnectW")
	procDisconnect       = modwinscard.NewProc("SCardDisconnect")
	procReconnect        = modwinscard.NewProc("SCardReconnect")
	procBeginTransaction = modwinscard.NewProc("SCardBeginTransaction")
	procEndTransaction   = modwinscard.NewProc("SCardEndTransaction")
	procStatus           = modwinscard.NewProc("SCardStatusW")
	procTransmit         = modwinscard.NewProc("SCardTransmit")
	procControl          = modwinscard.NewProc("SCardControl")
	procGetAttrib        = modwinscard.NewProc("SCardGetAttrib")
	procSetAttrib        = modwinscard.NewProc("SCardSetAttrib")

	dataT0Pci = modwinscard.NewProc("g_rgSCardT0Pci")
	dataT1Pci = modwinscard.NewProc("g_rgSCardT1Pci")
)

var scardIoReqT0 uintptr
var scardIoReqT1 uintptr

func init() {
	if err := dataT0Pci.Find(); err != nil {
		panic(err)
	}
	scardIoReqT0 = dataT0Pci.Addr()
	if err := dataT1Pci.Find(); err != nil {
		panic(err)
	}
	scardIoReqT1 = dataT1Pci.Addr()
}

func (e Error) Error() string {
	err := syscall.Errno(e)
	return fmt.Sprintf("scard: error(%x): %s", uintptr(e), err.Error())
}

func scardEstablishContext(scope Scope, reserved1, reserved2 uintptr) (uintptr, Error) {
	var ctx uintptr
	r, _, _ := procEstablishContext.Call(uintptr(scope), reserved1, reserved2, uintptr(unsafe.Pointer(&ctx)))
	return ctx, Error(r)
}

func scardIsValidContext(ctx uintptr) Error {
	r, _, _ := procIsValidContext.Call(ctx)
	return Error(r)
}

func scardCancel(ctx uintptr) Error {
	r, _, _ := procCancel.Call(ctx)
	return Error(r)
}

func scardReleaseContext(ctx uintptr) Error {
	r, _, _ := procReleaseContext.Call(ctx)
	return Error(r)
}

func scardListReaders(ctx uintptr, groups, buf unsafe.Pointer, bufLen uint32) (uint32, Error) {
	dwBufLen := uint32(bufLen)
	r, _, _ := procListReaders.Call(ctx, uintptr(groups), uintptr(buf), uintptr(unsafe.Pointer(&dwBufLen)))
	return dwBufLen, Error(r)
}

func scardListReaderGroups(ctx uintptr, buf unsafe.Pointer, bufLen uint32) (uint32, Error) {
	dwBufLen := uint32(bufLen)
	r, _, _ := procListReaderGroups.Call(ctx, uintptr(buf), uintptr(unsafe.Pointer(&dwBufLen)))
	return dwBufLen, Error(r)
}

func scardGetStatusChange(ctx uintptr, timeout uint32, states []scardReaderState) Error {
	r, _, _ := procGetStatusChange.Call(ctx, uintptr(timeout), uintptr(unsafe.Pointer(&states[0])), uintptr(len(states)))
	return Error(r)
}

func scardConnect(ctx uintptr, reader unsafe.Pointer, shareMode ShareMode, proto Protocol) (uintptr, Protocol, Error) {
	var handle uintptr
	var activeProto uint32

	r, _, _ := procConnect.Call(ctx, uintptr(reader), uintptr(shareMode), uintptr(proto), uintptr(unsafe.Pointer(&handle)), uintptr(unsafe.Pointer(&activeProto)))

	return handle, Protocol(activeProto), Error(r)
}

func scardDisconnect(card uintptr, d Disposition) Error {
	r, _, _ := procDisconnect.Call(card, uintptr(d))
	return Error(r)
}

func scardReconnect(card uintptr, mode ShareMode, proto Protocol, disp Disposition) (Protocol, Error) {
	var activeProtocol uint32
	r, _, _ := procReconnect.Call(card, uintptr(mode), uintptr(proto), uintptr(disp), uintptr(unsafe.Pointer(&activeProtocol)))
	return Protocol(activeProtocol), Error(r)
}

func scardBeginTransaction(card uintptr) Error {
	r, _, _ := procBeginTransaction.Call(card)
	return Error(r)
}

func scardEndTransaction(card uintptr, disp Disposition) Error {
	r, _, _ := procEndTransaction.Call(card, uintptr(disp))
	return Error(r)
}

func scardCardStatus(card uintptr) (string, State, Protocol, []byte, Error) {
	var state, proto uint32
	var atr [maxAtrSize]byte
	var atrLen = uint32(len(atr))

	reader := make(strbuf, maxReadername+1)
	readerLen := uint32(len(reader))

	r, _, _ := procStatus.Call(card, uintptr(reader.ptr()), uintptr(unsafe.Pointer(&readerLen)), uintptr(unsafe.Pointer(&state)), uintptr(unsafe.Pointer(&proto)), uintptr(unsafe.Pointer(&atr[0])), uintptr(unsafe.Pointer(&atrLen)))

	return decodestr(reader[:readerLen]), State(state), Protocol(proto), atr[:atrLen], Error(r)
}

func scardTransmit(card uintptr, proto Protocol, cmd []byte, rsp []byte) (uint32, Error) {
	var sendpci uintptr
	var rspLen = uint32(len(rsp))

	switch proto {
	case ProtocolT0:
		sendpci = scardIoReqT0
	case ProtocolT1:
		sendpci = scardIoReqT1
	default:
		panic("unknown protocol")
	}

	r, _, _ := procTransmit.Call(card, sendpci, uintptr(unsafe.Pointer(&cmd[0])), uintptr(len(cmd)), uintptr(0), uintptr(unsafe.Pointer(&rsp[0])), uintptr(unsafe.Pointer(&rspLen)))

	return rspLen, Error(r)
}

func scardControl(card uintptr, ioctl uint32, in, out []byte) (uint32, Error) {
	var ptrIn uintptr
	var outLen = uint32(len(out))

	if len(in) != 0 {
		ptrIn = uintptr(unsafe.Pointer(&in[0]))
	}

	r, _, _ := procControl.Call(card, uintptr(ioctl), ptrIn, uintptr(len(in)), uintptr(unsafe.Pointer(&out[0])), uintptr(len(out)), uintptr(unsafe.Pointer(&outLen)))
	return outLen, Error(r)
}

func scardGetAttrib(card uintptr, id Attrib, buf []byte) (uint32, Error) {
	var ptr uintptr

	if len(buf) != 0 {
		ptr = uintptr(unsafe.Pointer(&buf[0]))
	}

	bufLen := uint32(len(buf))
	r, _, _ := procGetAttrib.Call(card, uintptr(id), ptr, uintptr(unsafe.Pointer(&bufLen)))

	return bufLen, Error(r)
}

func scardSetAttrib(card uintptr, id Attrib, buf []byte) Error {
	r, _, _ := procSetAttrib.Call(card, uintptr(id), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return Error(r)
}

type scardReaderState struct {
	szReader       uintptr
	pvUserData     uintptr
	dwCurrentState uint32
	dwEventState   uint32
	cbAtr          uint32
	rgbAtr         [36]byte
}

func (rs *ReaderState) toSys() (scardReaderState, error) {
	var sys scardReaderState
	creader, err := encodestr(rs.Reader)
	if err != nil {
		return scardReaderState{}, err
	}
	sys.szReader = uintptr(creader.ptr())
	sys.dwCurrentState = uint32(rs.CurrentState)
	sys.cbAtr = uint32(len(rs.Atr))
	copy(sys.rgbAtr[:], rs.Atr)
	return sys, nil
}

func (rs *ReaderState) update(sys *scardReaderState) {
	rs.EventState = StateFlag(sys.dwEventState)
	if sys.cbAtr > 0 {
		rs.Atr = make([]byte, int(sys.cbAtr))
		copy(rs.Atr, sys.rgbAtr[:])
	}
}

type strbuf []uint16

func encodestr(s string) (strbuf, error) {
	utf16, err := syscall.UTF16FromString(s)
	return strbuf(utf16), err
}

func decodestr(buf strbuf) string {
	return syscall.UTF16ToString(buf)
}
