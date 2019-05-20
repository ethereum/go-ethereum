// Copyright 2014 The go-ethereum Authors
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

/*
Package whisper implements the Whisper PoC-1.

(https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec)

Whisper combines aspects of both DHTs and datagram messaging systems (e.g. UDP).
As such it may be likened and compared to both, not dissimilar to the
matter/energy duality (apologies to physicists for the blatant abuse of a
fundamental and beautiful natural principle).

Whisper is a pure identity-based messaging system. Whisper provides a low-level
(non-application-specific) but easily-accessible API without being based upon
or prejudiced by the low-level hardware attributes and characteristics,
particularly the notion of singular endpoints.
*/
package whisper
