// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package logger

import "os"

func ExampleLogger() {
	logger := NewLogger("TAG")
	logger.Infoln("so awesome")            // prints [TAG] so awesome
	logger.Infof("this %q is raw", "coin") // prints [TAG] this "coin" is raw
}

func ExampleLogSystem() {
	filename := "test.log"
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	fileLog := NewStdLogSystem(file, 0, WarnLevel)
	AddLogSystem(fileLog)

	stdoutLog := NewStdLogSystem(os.Stdout, 0, WarnLevel)
	AddLogSystem(stdoutLog)

	NewLogger("TAG").Warnln("reactor meltdown") // writes to both logs
}
