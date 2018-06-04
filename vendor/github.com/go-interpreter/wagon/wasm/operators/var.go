// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

var (
	GetLocal  = newPolymorphicOp(0x20, "get_local")
	SetLocal  = newPolymorphicOp(0x21, "set_local")
	TeeLocal  = newPolymorphicOp(0x22, "tee_local")
	GetGlobal = newPolymorphicOp(0x23, "get_global")
	SetGlobal = newPolymorphicOp(0x24, "set_global")
)
