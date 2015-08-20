// Copyright 2014 shiena Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*

The ansicolor command colors a console text by ANSI escape sequence like wac.

    $ go get github.com/shiena/ansicolor/ansicolor

See also:
    https://github.com/aslakhellesoy/wac

*/
package main

import (
	"io"
	"os"

	"github.com/shiena/ansicolor"
)

func main() {
	w := ansicolor.NewAnsiColorWriter(os.Stdout)
	io.Copy(w, os.Stdin)
}
