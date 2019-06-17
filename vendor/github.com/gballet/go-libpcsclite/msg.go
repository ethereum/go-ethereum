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
	"net"
)

/**
 * @brief Wrapper for the MessageSend() function.
 *
 * Called by clients to send messages to the server.
 * The parameters \p command and \p data are set in the \c sharedSegmentMsg
 * struct in order to be sent.
 *
 * @param[in] command Command to be sent.
 * @param[in] dwClientID Client socket handle.
 * @param[in] size Size of the message (\p data).
 * @param[in] data_void Data to be sent.
 *
 * @return Same error codes as MessageSend().
 */
func messageSendWithHeader(command uint32, conn net.Conn, data []byte) error {
	/* Translate header into bytes */
	msgData := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint32(msgData[4:], command)
	binary.LittleEndian.PutUint32(msgData, uint32(len(data)))

	/* Copy payload */
	copy(msgData[8:], data)

	_, err := conn.Write(msgData)
	return err
}

// clientSetupSession prepares a communication channel for the client to talk to the server.
// This is called by the application to create a socket for local IPC with the
// server. The socket is associated to the file \c PCSCLITE_CSOCK_NAME.
/*
 * @param[out] pdwClientID Client Connection ID.
 *
 * @retval 0 Success.
 * @retval -1 Can not create the socket.
 * @retval -1 The socket can not open a connection.
 * @retval -1 Can not set the socket to non-blocking.
 */
func clientSetupSession(daemonPath string) (net.Conn, error) {
	path := PCSCDSockName
	if len(daemonPath) > 0 {
		path = daemonPath
	}
	return net.Dial("unix", path)
}
