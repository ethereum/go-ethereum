// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operators

var (
	Drop   = newPolymorphicOp(0x1a, "drop")
	Select = newPolymorphicOp(0x1b, "select")
)
