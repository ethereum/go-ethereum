// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

var (
	Call         = newPolymorphicOp(0x10, "call")
	CallIndirect = newPolymorphicOp(0x11, "call_indirect")
)
