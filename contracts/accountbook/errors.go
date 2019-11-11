// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package accountbook

import "errors"

// ErrNotEnoughDeposit is returned if the cheque drawer doesn't have enough
// balance to spend. Whenever drawee receives this error, it should emit a
// cash operation as soon as possible.
var ErrNotEnoughDeposit = errors.New("deposit is not enough")

// StaleChequeError wraps a error msg and the evidence for a stale cheque.
//
// Cheque drawer can sign the stale cheques deliberately or indeliberately.
// E.G. If the cheque db of drawer is missing, it can lead to a indeliberate
// stale cheque.
type StaleChequeError struct {
	Msg      string
	Evidence *Cheque // The latest received cheque can be used as an evidence
}

func (err *StaleChequeError) Error() string {
	return err.Msg
}
