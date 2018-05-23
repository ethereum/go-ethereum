// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
Package base58 provides base58-check encoding.
The alphabet is modifyiable for

Base58 Usage

To decode a base58 string:

 rawData := base58.Base58Decode(encodedData)

Similarly, to encode the same data:

 encodedData := base58.Base58Encode(rawData)

*/
package base58
