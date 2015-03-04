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

package main

import (
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/xeth"
)

func ExecJsFile(ethereum *eth.Ethereum, InputFile string) {
	file, err := os.Open(InputFile)
	if err != nil {
		clilogger.Fatalln(err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		clilogger.Fatalln(err)
	}
	re := javascript.NewJSRE(xeth.New(ethereum))
	re.Run(string(content))
}
