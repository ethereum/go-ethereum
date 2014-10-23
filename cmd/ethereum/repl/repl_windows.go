// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package ethrepl

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (self *JSRepl) read() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf(self.prompt)
		str, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Error reading input", err)
		} else {
			if (string(str) == "exit") {
				self.Stop()
				break
			} else {
				self.parseInput(string(str))
			}
		}
	}
}

func addHistory(s string) {
}

func printColored(outputVal string) {
	for ; outputVal != "" ; {
		codePart := ""
		if (strings.HasPrefix(outputVal, "\033[32m")) {
			codePart = "\033[32m"
			changeColor(2)
		}
		if (strings.HasPrefix(outputVal, "\033[1m\033[30m")) {
			codePart = "\033[1m\033[30m"
			changeColor(8)
		}
		if (strings.HasPrefix(outputVal, "\033[31m")) {
			codePart = "\033[31m"
			changeColor(red)
		}
		if (strings.HasPrefix(outputVal, "\033[35m")) {
			codePart = "\033[35m"
			changeColor(5)
		}
		if (strings.HasPrefix(outputVal, "\033[0m")) {
			codePart = "\033[0m"
			resetColorful()
		}
		textPart := outputVal[len(codePart):len(outputVal)]
		index := strings.Index(textPart, "\033")
		if index == -1 {
			outputVal = ""
		} else {
			outputVal = textPart[index:len(textPart)]
			textPart = textPart[0:index]
		}
		fmt.Printf("%v", textPart)
	}
}

func (self *JSRepl) PrintValue(v interface{}) {
	method, _ := self.re.Vm.Get("prettyPrint")
	v, err := self.re.Vm.ToValue(v)
	if err == nil {
		val, err := method.Call(method, v)
		if err == nil {
			printColored(fmt.Sprintf("%v", val))
		}
	}
}
