// Copyright 2013 Google Inc.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package usb

import (
	"fmt"
)

type BCD uint16

const (
	USB_2_0 BCD = 0x0200
	USB_1_1 BCD = 0x0110
	USB_1_0 BCD = 0x0100
)

func (d BCD) Int() (i int) {
	ten := 1
	for o := uint(0); o < 4; o++ {
		n := ((0xF << (o * 4)) & d) >> (o * 4)
		i += int(n) * ten
		ten *= 10
	}
	return
}

func (d BCD) String() string {
	return fmt.Sprintf("%02x.%02x", int(d>>8), int(d&0xFF))
}

type ID uint16

func (id ID) String() string {
	return fmt.Sprintf("%04x", int(id))
}
