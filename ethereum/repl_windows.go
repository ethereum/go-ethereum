package main

import (
	"bufio"
	"fmt"
	"os"
)

func (self *JSRepl) read() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf(self.prompt)
		str, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Error reading input", err)
		} else {
			self.parseInput(string(str))
		}
	}
}
