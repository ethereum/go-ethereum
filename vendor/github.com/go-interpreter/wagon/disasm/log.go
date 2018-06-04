// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disasm

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	logger  *log.Logger
	logging bool
)

func SetDebugMode(l bool) {
	w := ioutil.Discard
	logging = l

	if l {
		w = os.Stderr
	}

	logger = log.New(w, "", log.Lshortfile)
	logger.SetFlags(log.Lshortfile)

}

func init() {
	SetDebugMode(false)
}
