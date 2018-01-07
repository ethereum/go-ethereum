// +build darwin

package scard

// #cgo LDFLAGS: -framework PCSC
// #cgo CFLAGS: -I /usr/include
// #include <stdlib.h>
// #include <PCSC/winscard.h>
// #include <PCSC/wintypes.h>
import "C"

import (
	"unsafe"
)

func (e Error) Error() string {
	return "scard: " + C.GoString(C.pcsc_stringify_error(C.int32_t(e)))
}

// Version returns the libpcsclite version string
func Version() string {
	return C.PCSCLITE_VERSION_NUMBER
}

func scardEstablishContext(scope Scope, reserved1, reserved2 uintptr) (uintptr, Error) {
	var ctx C.SCARDCONTEXT
	r := C.SCardEstablishContext(C.uint32_t(scope), unsafe.Pointer(reserved1), unsafe.Pointer(reserved2), &ctx)
	return uintptr(ctx), Error(r)
}

func scardIsValidContext(ctx uintptr) Error {
	r := C.SCardIsValidContext(C.SCARDCONTEXT(ctx))
	return Error(r)
}

func scardCancel(ctx uintptr) Error {
	r := C.SCardCancel(C.SCARDCONTEXT(ctx))
	return Error(r)
}

func scardReleaseContext(ctx uintptr) Error {
	r := C.SCardReleaseContext(C.SCARDCONTEXT(ctx))
	return Error(r)
}

func scardListReaders(ctx uintptr, groups, buf unsafe.Pointer, bufLen uint32) (uint32, Error) {
	dwBufLen := C.uint32_t(bufLen)
	r := C.SCardListReaders(C.SCARDCONTEXT(ctx), (C.LPCSTR)(groups), (C.LPSTR)(buf), &dwBufLen)
	return uint32(dwBufLen), Error(r)
}

func scardListReaderGroups(ctx uintptr, buf unsafe.Pointer, bufLen uint32) (uint32, Error) {
	dwBufLen := C.uint32_t(bufLen)
	r := C.SCardListReaderGroups(C.SCARDCONTEXT(ctx), (C.LPSTR)(buf), &dwBufLen)
	return uint32(dwBufLen), Error(r)
}

func scardGetStatusChange(ctx uintptr, timeout uint32, states []scardReaderState) Error {
	// In darwin, the LPSCARD_READERSTATE_A has 1 byte alignment and hence
	// has no trailing padding. Go does add 3 bytes of padding (on both 32
	// and 64 bits), so we pack an array manually instead.
	const size = int(unsafe.Sizeof(states[0])) - 3
	buf := make([]byte, size*len(states))
	for i, _ := range states {
		copy(buf[i*size:(i+1)*size], (*(*[size]byte)(unsafe.Pointer(&states[i])))[:])
	}
	r := C.SCardGetStatusChange(C.SCARDCONTEXT(ctx), C.uint32_t(timeout), (C.LPSCARD_READERSTATE_A)(unsafe.Pointer(&buf[0])), C.uint32_t(len(states)))
	for i, _ := range states {
		copy((*(*[size]byte)(unsafe.Pointer(&states[i])))[:], buf[i*size:(i+1)*size])
	}
	return Error(r)
}

func scardConnect(ctx uintptr, reader unsafe.Pointer, shareMode ShareMode, proto Protocol) (uintptr, Protocol, Error) {
	var handle C.SCARDHANDLE
	var activeProto C.uint32_t

	r := C.SCardConnect(C.SCARDCONTEXT(ctx), C.LPCSTR(reader), C.uint32_t(shareMode), C.uint32_t(proto), &handle, &activeProto)

	return uintptr(handle), Protocol(activeProto), Error(r)
}

func scardDisconnect(card uintptr, d Disposition) Error {
	r := C.SCardDisconnect(C.SCARDHANDLE(card), C.uint32_t(d))
	return Error(r)
}

func scardReconnect(card uintptr, mode ShareMode, proto Protocol, disp Disposition) (Protocol, Error) {
	var activeProtocol C.uint32_t
	r := C.SCardReconnect(C.SCARDHANDLE(card), C.uint32_t(mode), C.uint32_t(proto), C.uint32_t(disp), &activeProtocol)
	return Protocol(activeProtocol), Error(r)
}

func scardBeginTransaction(card uintptr) Error {
	r := C.SCardBeginTransaction(C.SCARDHANDLE(card))
	return Error(r)
}

func scardEndTransaction(card uintptr, disp Disposition) Error {
	r := C.SCardEndTransaction(C.SCARDHANDLE(card), C.uint32_t(disp))
	return Error(r)
}

func scardCardStatus(card uintptr) (string, State, Protocol, []byte, Error) {
	var readerBuf [C.MAX_READERNAME + 1]byte
	var readerLen = C.uint32_t(len(readerBuf))
	var state, proto C.uint32_t
	var atr [maxAtrSize]byte
	var atrLen = C.uint32_t(len(atr))

	r := C.SCardStatus(C.SCARDHANDLE(card), (C.LPSTR)(unsafe.Pointer(&readerBuf[0])), &readerLen, &state, &proto, (*C.uchar)(&atr[0]), &atrLen)

	return decodestr(readerBuf[:readerLen]), State(state), Protocol(proto), atr[:atrLen], Error(r)
}

func scardTransmit(card uintptr, proto Protocol, cmd []byte, rsp []byte) (uint32, Error) {
	var sendpci C.SCARD_IO_REQUEST
	var recvpci C.SCARD_IO_REQUEST
	var rspLen = C.uint32_t(len(rsp))

	switch proto {
	case ProtocolT0, ProtocolT1:
		sendpci.dwProtocol = C.uint32_t(proto)
	default:
		panic("unknown protocol")
	}
	sendpci.cbPciLength = C.sizeof_SCARD_IO_REQUEST

	r := C.SCardTransmit(C.SCARDHANDLE(card), &sendpci, (*C.uchar)(&cmd[0]), C.uint32_t(len(cmd)), &recvpci, (*C.uchar)(&rsp[0]), &rspLen)

	return uint32(rspLen), Error(r)
}

func scardControl(card uintptr, ioctl uint32, in, out []byte) (uint32, Error) {
	var ptrIn unsafe.Pointer
	var outLen = C.uint32_t(len(out))

	if len(in) != 0 {
		ptrIn = unsafe.Pointer(&in[0])
	}

	r := C.SCardControl(C.SCARDHANDLE(card), C.uint32_t(ioctl), ptrIn, C.uint32_t(len(in)), unsafe.Pointer(&out[0]), C.uint32_t(len(out)), &outLen)
	return uint32(outLen), Error(r)
}

func scardGetAttrib(card uintptr, id Attrib, buf []byte) (uint32, Error) {
	var ptr *C.uint8_t

	if len(buf) != 0 {
		ptr = (*C.uint8_t)(&buf[0])
	}

	bufLen := C.uint32_t(len(buf))
	r := C.SCardGetAttrib(C.SCARDHANDLE(card), C.uint32_t(id), ptr, &bufLen)

	return uint32(bufLen), Error(r)
}

func scardSetAttrib(card uintptr, id Attrib, buf []byte) Error {
	r := C.SCardSetAttrib(C.SCARDHANDLE(card), C.uint32_t(id), ((*C.uint8_t)(&buf[0])), C.uint32_t(len(buf)))
	return Error(r)
}

type strbuf []byte

func encodestr(s string) (strbuf, error) {
	buf := strbuf(s + "\x00")
	return buf, nil
}

func decodestr(buf strbuf) string {
	if len(buf) == 0 {
		return ""
	}

	if buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}

	return string(buf)
}

type scardReaderState struct {
	szReader       uintptr
	pvUserData     uintptr
	dwCurrentState uint32
	dwEventState   uint32
	cbAtr          uint32
	rgbAtr         [33]byte
}

var pinned = map[string]*strbuf{}

func (rs *ReaderState) toSys() (scardReaderState, error) {
	var sys scardReaderState

	creader, err := encodestr(rs.Reader)
	if err != nil {
		return scardReaderState{}, err
	}
	pinned[rs.Reader] = &creader
	sys.szReader = uintptr(creader.ptr())
	sys.dwCurrentState = uint32(rs.CurrentState)
	sys.cbAtr = uint32(len(rs.Atr))
	for i, v := range rs.Atr {
		sys.rgbAtr[i] = byte(v)
	}
	return sys, nil
}

func (rs *ReaderState) update(sys *scardReaderState) {
	rs.EventState = StateFlag(sys.dwEventState)
	if sys.cbAtr > 0 {
		rs.Atr = make([]byte, int(sys.cbAtr))
		for i := 0; i < int(sys.cbAtr); i++ {
			rs.Atr[i] = byte(sys.rgbAtr[i])
		}
	}
}
