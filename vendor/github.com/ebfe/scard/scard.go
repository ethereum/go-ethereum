// Package scard provides bindings to the PC/SC API.
package scard

import (
	"time"
	"unsafe"
)

type CardStatus struct {
	Reader         string
	State          State
	ActiveProtocol Protocol
	Atr            []byte
}

type ReaderState struct {
	Reader       string
	UserData     interface{}
	CurrentState StateFlag
	EventState   StateFlag
	Atr          []byte
}

type Context struct {
	ctx uintptr
}

type Card struct {
	handle         uintptr
	activeProtocol Protocol
}

// wraps SCardEstablishContext
func EstablishContext() (*Context, error) {
	ctx, r := scardEstablishContext(ScopeSystem, 0, 0)
	if r != ErrSuccess {
		return nil, r
	}

	return &Context{ctx: ctx}, nil
}

// wraps SCardIsValidContext
func (ctx *Context) IsValid() (bool, error) {
	r := scardIsValidContext(ctx.ctx)
	switch r {
	case ErrSuccess:
		return true, nil
	case ErrInvalidHandle:
		return false, nil
	default:
		return false, r
	}
}

// wraps SCardCancel
func (ctx *Context) Cancel() error {
	r := scardCancel(ctx.ctx)
	if r != ErrSuccess {
		return r
	}
	return nil
}

// wraps SCardReleaseContext
func (ctx *Context) Release() error {
	r := scardReleaseContext(ctx.ctx)
	if r != ErrSuccess {
		return r
	}
	return nil
}

// wraps SCardListReaders
func (ctx *Context) ListReaders() ([]string, error) {
	needed, r := scardListReaders(ctx.ctx, nil, nil, 0)
	if r != ErrSuccess {
		return nil, r
	}

	buf := make(strbuf, needed)
	n, r := scardListReaders(ctx.ctx, nil, buf.ptr(), uint32(len(buf)))
	if r != ErrSuccess {
		return nil, r
	}
	return decodemstr(buf[:n]), nil
}

// wraps SCardListReaderGroups
func (ctx *Context) ListReaderGroups() ([]string, error) {
	needed, r := scardListReaderGroups(ctx.ctx, nil, 0)
	if r != ErrSuccess {
		return nil, r
	}

	buf := make(strbuf, needed)
	n, r := scardListReaderGroups(ctx.ctx, buf.ptr(), uint32(len(buf)))
	if r != ErrSuccess {
		return nil, r
	}
	return decodemstr(buf[:n]), nil
}

// wraps SCardGetStatusChange
func (ctx *Context) GetStatusChange(readerStates []ReaderState, timeout time.Duration) error {

	dwTimeout := durationToTimeout(timeout)
	states := make([]scardReaderState, len(readerStates))

	for i := range readerStates {
		var err error
		states[i], err = readerStates[i].toSys()
		if err != nil {
			return err
		}
	}

	r := scardGetStatusChange(ctx.ctx, dwTimeout, states)
	if r != ErrSuccess {
		return r
	}

	for i := range readerStates {
		(&readerStates[i]).update(&states[i])
	}

	return nil
}

// wraps SCardConnect
func (ctx *Context) Connect(reader string, mode ShareMode, proto Protocol) (*Card, error) {
	creader, err := encodestr(reader)
	if err != nil {
		return nil, err
	}
	handle, activeProtocol, r := scardConnect(ctx.ctx, creader.ptr(), mode, proto)
	if r != ErrSuccess {
		return nil, r
	}
	return &Card{handle: handle, activeProtocol: activeProtocol}, nil
}

// wraps SCardDisconnect
func (card *Card) Disconnect(d Disposition) error {
	r := scardDisconnect(card.handle, d)
	if r != ErrSuccess {
		return r
	}
	return nil
}

// wraps SCardReconnect
func (card *Card) Reconnect(mode ShareMode, proto Protocol, disp Disposition) error {
	activeProtocol, r := scardReconnect(card.handle, mode, proto, disp)
	if r != ErrSuccess {
		return r
	}
	card.activeProtocol = activeProtocol
	return nil
}

// wraps SCardBeginTransaction
func (card *Card) BeginTransaction() error {
	r := scardBeginTransaction(card.handle)
	if r != ErrSuccess {
		return r
	}
	return nil
}

// wraps SCardEndTransaction
func (card *Card) EndTransaction(disp Disposition) error {
	r := scardEndTransaction(card.handle, disp)
	if r != ErrSuccess {
		return r
	}
	return nil
}

// wraps SCardStatus
func (card *Card) Status() (*CardStatus, error) {
	reader, state, proto, atr, err := scardCardStatus(card.handle)
	if err != ErrSuccess {
		return nil, err
	}
	return &CardStatus{Reader: reader, State: state, ActiveProtocol: proto, Atr: atr}, nil
}

// wraps SCardTransmit
func (card *Card) Transmit(cmd []byte) ([]byte, error) {
	rsp := make([]byte, maxBufferSizeExtended)
	rspLen, err := scardTransmit(card.handle, card.activeProtocol, cmd, rsp)
	if err != ErrSuccess {
		return nil, err
	}
	return rsp[:rspLen], nil
}

// wraps SCardControl
func (card *Card) Control(ioctl uint32, in []byte) ([]byte, error) {
	var out [0xffff]byte
	outLen, err := scardControl(card.handle, ioctl, in, out[:])
	if err != ErrSuccess {
		return nil, err
	}
	return out[:outLen], nil
}

// wraps SCardGetAttrib
func (card *Card) GetAttrib(id Attrib) ([]byte, error) {
	needed, err := scardGetAttrib(card.handle, id, nil)
	if err != ErrSuccess {
		return nil, err
	}

	var attrib = make([]byte, needed)
	n, err := scardGetAttrib(card.handle, id, attrib)
	if err != ErrSuccess {
		return nil, err
	}
	return attrib[:n], nil
}

// wraps SCardSetAttrib
func (card *Card) SetAttrib(id Attrib, data []byte) error {
	err := scardSetAttrib(card.handle, id, data)
	if err != ErrSuccess {
		return err
	}
	return nil
}

func durationToTimeout(timeout time.Duration) uint32 {
	switch {
	case timeout < 0:
		return infiniteTimeout
	case timeout > time.Duration(infiniteTimeout)*time.Millisecond:
		return infiniteTimeout - 1
	default:
		return uint32(timeout / time.Millisecond)
	}
}

func (buf strbuf) ptr() unsafe.Pointer {
	return unsafe.Pointer(&buf[0])
}

func (buf strbuf) split() []strbuf {
	var chunks []strbuf
	for len(buf) > 0 && buf[0] != 0 {
		i := 0
		for i = range buf {
			if buf[i] == 0 {
				break
			}
		}
		chunks = append(chunks, buf[:i+1])
		buf = buf[i+1:]
	}

	return chunks
}

func encodemstr(strings ...string) (strbuf, error) {
	var buf strbuf
	for _, s := range strings {
		utf16, err := encodestr(s)
		if err != nil {
			return nil, err
		}
		buf = append(buf, utf16...)
	}
	buf = append(buf, 0)
	return buf, nil
}

func decodemstr(buf strbuf) []string {
	var strings []string
	for _, chunk := range buf.split() {
		strings = append(strings, decodestr(chunk))
	}
	return strings
}
