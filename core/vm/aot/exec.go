// Copyright (c) 2018 Timo Savola. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aot

func Exec(textBase, stackLimit, memoryBase, stackPtr uintptr) (int, int)
func ImportTrapHandler() uint64
func ImportGrowMemory() uint64
