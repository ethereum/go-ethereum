// BSD 3-Clause License
//
// Copyright (c) 2019, Guillaume Ballet
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
//
// * Neither the name of the copyright holder nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package pcsc

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"unsafe"
)

// Client contains all the information needed to establish
// and maintain a connection to the deamon/card.
type Client struct {
	conn net.Conn

	minor uint32
	major uint32

	ctx uint32

	mutex sync.Mutex

	readerStateDescriptors [MaxReaderStateDescriptors]ReaderState
}

// EstablishContext asks the PCSC daemon to create a context
// handle for further communication with connected cards and
// readers.
func EstablishContext(scope uint32) (*Client, error) {
	client := &Client{}

	conn, err := clientSetupSession()
	if err != nil {
		return nil, err
	}
	client.conn = conn

	payload := make([]byte, 12)
	response := make([]byte, 12)

	var code uint32
	var minor uint32
	for minor = ProtocolVersionMinor; minor <= ProtocolVersionMinor+1; minor++ {
		/* Exchange version information */
		binary.LittleEndian.PutUint32(payload, ProtocolVersionMajor)
		binary.LittleEndian.PutUint32(payload[4:], minor)
		binary.LittleEndian.PutUint32(payload[8:], SCardSuccess.Code())
		err = messageSendWithHeader(CommandVersion, conn, payload)
		if err != nil {
			return nil, err
		}
		n, err := conn.Read(response)
		if err != nil {
			return nil, err
		}
		if n != len(response) {
			return nil, fmt.Errorf("invalid response length: expected %d, got %d", len(response), n)
		}
		code = binary.LittleEndian.Uint32(response[8:])
		if code != SCardSuccess.Code() {
			continue
		}
		client.major = binary.LittleEndian.Uint32(response)
		client.minor = binary.LittleEndian.Uint32(response[4:])
		if client.major != ProtocolVersionMajor || client.minor != minor {
			continue
		}
		break
	}

	if code != SCardSuccess.Code() {
		return nil, fmt.Errorf("invalid response code: expected %d, got %d (%v)", SCardSuccess, code, ErrorCode(code).Error())
	}
	if client.major != ProtocolVersionMajor || (client.minor != minor && client.minor+1 != minor) {
		return nil, fmt.Errorf("invalid version found: expected %d.%d, got %d.%d", ProtocolVersionMajor, ProtocolVersionMinor, client.major, client.minor)
	}

	/* Establish the context proper */
	binary.LittleEndian.PutUint32(payload, scope)
	binary.LittleEndian.PutUint32(payload[4:], 0)
	binary.LittleEndian.PutUint32(payload[8:], SCardSuccess.Code())
	err = messageSendWithHeader(SCardEstablishContext, conn, payload)
	if err != nil {
		return nil, err
	}
	response = make([]byte, 12)
	n, err := conn.Read(response)
	if err != nil {
		return nil, err
	}
	if n != len(response) {
		return nil, fmt.Errorf("invalid response length: expected %d, got %d", len(response), n)
	}
	code = binary.LittleEndian.Uint32(response[8:])
	if code != SCardSuccess.Code() {
		return nil, fmt.Errorf("invalid response code: expected %d, got %d (%v)", SCardSuccess, code, ErrorCode(code).Error())
	}
	client.ctx = binary.LittleEndian.Uint32(response[4:])

	return client, nil
}

// ReleaseContext tells the daemon that the client will no longer
// need the context.
func (client *Client) ReleaseContext() error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	data := [8]byte{}
	binary.LittleEndian.PutUint32(data[:], client.ctx)
	binary.LittleEndian.PutUint32(data[4:], SCardSuccess.Code())
	err := messageSendWithHeader(SCardReleaseContext, client.conn, data[:])
	if err != nil {
		return err
	}
	total := 0
	for total < len(data) {
		n, err := client.conn.Read(data[total:])
		if err != nil {
			return err
		}
		total += n
	}
	code := binary.LittleEndian.Uint32(data[4:])
	if code != SCardSuccess.Code() {
		return fmt.Errorf("invalid return code: %x, %v", code, ErrorCode(code).Error())
	}

	return nil
}

// Constants related to the reader state structure
const (
	ReaderStateNameLength       = 128
	ReaderStateMaxAtrSizeLength = 33
	// NOTE: ATR is 32-byte aligned in the C version, which means it's
	// actually 36 byte long and not 33.
	ReaderStateDescriptorLength = ReaderStateNameLength + ReaderStateMaxAtrSizeLength + 5*4 + 3

	MaxReaderStateDescriptors = 16
)

// ReaderState represent the state of a single reader, as reported
// by the PCSC daemon.
type ReaderState struct {
	Name          string /* reader name */
	eventCounter  uint32 /* number of card events */
	readerState   uint32 /* SCARD_* bit field */
	readerSharing uint32 /* PCSCLITE_SHARING_* sharing status */

	cardAtr       [ReaderStateMaxAtrSizeLength]byte /* ATR */
	cardAtrLength uint32                            /* ATR length */
	cardProtocol  uint32                            /* SCARD_PROTOCOL_* value */
}

func getReaderState(data []byte) (ReaderState, error) {
	ret := ReaderState{}
	if len(data) < ReaderStateDescriptorLength {
		return ret, fmt.Errorf("could not unmarshall data of length %d < %d", len(data), ReaderStateDescriptorLength)
	}

	ret.Name = string(data[:ReaderStateNameLength])
	ret.eventCounter = binary.LittleEndian.Uint32(data[unsafe.Offsetof(ret.eventCounter):])
	ret.readerState = binary.LittleEndian.Uint32(data[unsafe.Offsetof(ret.readerState):])
	ret.readerSharing = binary.LittleEndian.Uint32(data[unsafe.Offsetof(ret.readerSharing):])
	copy(ret.cardAtr[:], data[unsafe.Offsetof(ret.cardAtr):unsafe.Offsetof(ret.cardAtr)+ReaderStateMaxAtrSizeLength])
	ret.cardAtrLength = binary.LittleEndian.Uint32(data[unsafe.Offsetof(ret.cardAtrLength):])
	ret.cardProtocol = binary.LittleEndian.Uint32(data[unsafe.Offsetof(ret.cardProtocol):])

	return ret, nil
}

// ListReaders gets the list of readers from the daemon
func (client *Client) ListReaders() ([]string, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	err := messageSendWithHeader(CommandGetReaderState, client.conn, []byte{})
	if err != nil {
		return nil, err
	}
	response := make([]byte, ReaderStateDescriptorLength*MaxReaderStateDescriptors)
	total := 0
	for total < len(response) {
		n, err := client.conn.Read(response[total:])
		if err != nil {
			return nil, err
		}
		total += n
	}

	var names []string
	for i := range client.readerStateDescriptors {
		desc, err := getReaderState(response[i*ReaderStateDescriptorLength:])
		if err != nil {
			return nil, err
		}
		client.readerStateDescriptors[i] = desc
		if desc.Name[0] == 0 {
			break
		}
		names = append(names, desc.Name)
	}

	return names, nil
}

// Offsets into the Connect request/response packet
const (
	SCardConnectReaderNameOffset        = 4
	SCardConnectShareModeOffset         = SCardConnectReaderNameOffset + ReaderStateNameLength
	SCardConnectPreferredProtocolOffset = SCardConnectShareModeOffset + 4
	SCardConnectReturnValueOffset       = SCardConnectPreferredProtocolOffset + 12
)

// Card represents the connection to a card
type Card struct {
	handle      uint32
	activeProto uint32
	client      *Client
}

// Connect asks the daemon to connect to the card
func (client *Client) Connect(name string, shareMode uint32, preferredProtocol uint32) (*Card, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	request := make([]byte, ReaderStateNameLength+4*6)
	binary.LittleEndian.PutUint32(request, client.ctx)
	copy(request[SCardConnectReaderNameOffset:], []byte(name))
	binary.LittleEndian.PutUint32(request[SCardConnectShareModeOffset:], shareMode)
	binary.LittleEndian.PutUint32(request[SCardConnectPreferredProtocolOffset:], preferredProtocol)
	binary.LittleEndian.PutUint32(request[SCardConnectReturnValueOffset:], SCardSuccess.Code())

	err := messageSendWithHeader(SCardConnect, client.conn, request)
	if err != nil {
		return nil, err
	}
	response := make([]byte, ReaderStateNameLength+4*6)
	total := 0
	for total < len(response) {
		n, err := client.conn.Read(response[total:])
		if err != nil {
			return nil, err
		}
		// fmt.Println("total, n", total, n, response)
		total += n
	}
	code := binary.LittleEndian.Uint32(response[148:])
	if code != SCardSuccess.Code() {
		return nil, fmt.Errorf("invalid return code: %x (%v)", code, ErrorCode(code).Error())
	}
	handle := binary.LittleEndian.Uint32(response[140:])
	active := binary.LittleEndian.Uint32(response[SCardConnectPreferredProtocolOffset:])

	return &Card{handle: handle, activeProto: active, client: client}, nil
}

/**
* @brief contained in \ref SCARD_TRANSMIT Messages.
*
* These data are passed throw the field \c sharedSegmentMsg.data.
 */
type transmit struct {
	hCard             uint32
	ioSendPciProtocol uint32
	ioSendPciLength   uint32
	cbSendLength      uint32
	ioRecvPciProtocol uint32
	ioRecvPciLength   uint32
	pcbRecvLength     uint32
	rv                uint32
}

// SCardIoRequest contains the info needed for performing an IO request
type SCardIoRequest struct {
	proto  uint32
	length uint32
}

const (
	TransmitRequestLength = 32
)

// Transmit sends request data to a card and returns the response
func (card *Card) Transmit(adpu []byte) ([]byte, *SCardIoRequest, error) {
	card.client.mutex.Lock()
	defer card.client.mutex.Unlock()

	request := [TransmitRequestLength]byte{}
	binary.LittleEndian.PutUint32(request[:], card.handle)
	binary.LittleEndian.PutUint32(request[4:] /*card.activeProto*/, 2)
	binary.LittleEndian.PutUint32(request[8:], 8)
	binary.LittleEndian.PutUint32(request[12:], uint32(len(adpu)))
	binary.LittleEndian.PutUint32(request[16:], 0)
	binary.LittleEndian.PutUint32(request[20:], 0)
	binary.LittleEndian.PutUint32(request[24:], 0x10000)
	binary.LittleEndian.PutUint32(request[28:], SCardSuccess.Code())
	err := messageSendWithHeader(SCardTransmit, card.client.conn, request[:])
	if err != nil {
		return nil, nil, err
	}
	// Add the ADPU payload after the transmit descriptor
	n, err := card.client.conn.Write(adpu)
	if err != nil {
		return nil, nil, err
	}
	if n != len(adpu) {
		return nil, nil, fmt.Errorf("Invalid number of bytes written: expected %d, got %d", len(adpu), n)
	}
	response := [TransmitRequestLength]byte{}
	total := 0
	for total < len(response) {
		n, err = card.client.conn.Read(response[total:])
		if err != nil {
			return nil, nil, err
		}
		total += n
	}

	code := binary.LittleEndian.Uint32(response[28:])
	if code != SCardSuccess.Code() {
		return nil, nil, fmt.Errorf("invalid return code: %x (%v)", code, ErrorCode(code).Error())
	}

	// Recover the response data
	recvProto := binary.LittleEndian.Uint32(response[16:])
	recvLength := binary.LittleEndian.Uint32(response[20:])
	recv := &SCardIoRequest{proto: recvProto, length: recvLength}
	recvLength = binary.LittleEndian.Uint32(response[24:])
	recvData := make([]byte, recvLength)
	total = 0
	for uint32(total) < recvLength {
		n, err := card.client.conn.Read(recvData[total:])
		if err != nil {
			return nil, nil, err
		}
		total += n
	}

	return recvData, recv, nil
}

// Disconnect tells the PCSC daemon that the client is no longer
// interested in communicating with the card.
func (card *Card) Disconnect(disposition uint32) error {
	card.client.mutex.Lock()
	defer card.client.mutex.Unlock()

	data := [12]byte{}
	binary.LittleEndian.PutUint32(data[:], card.handle)
	binary.LittleEndian.PutUint32(data[4:], disposition)
	binary.LittleEndian.PutUint32(data[8:], SCardSuccess.Code())
	err := messageSendWithHeader(SCardDisConnect, card.client.conn, data[:])
	if err != nil {
		return err
	}
	total := 0
	for total < len(data) {
		n, err := card.client.conn.Read(data[total:])
		if err != nil {
			return err
		}
		total += n
	}
	code := binary.LittleEndian.Uint32(data[8:])
	if code != SCardSuccess.Code() {
		return fmt.Errorf("invalid return code: %x (%v)", code, ErrorCode(code).Error())
	}

	return nil
}
