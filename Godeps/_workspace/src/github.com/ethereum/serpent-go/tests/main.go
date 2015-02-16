package main

import (
	"fmt"

	"github.com/ethereum/serpent-go"
)

func main() {
	out, _ := serpent.Compile(`
// Namecoin
if !contract.storage[msg.data[0]]: # Is the key not yet taken?
    # Then take it!
    contract.storage[msg.data[0]] = msg.data[1]
    return(1)
else:
    return(0) // Otherwise do nothing
	`)

	fmt.Printf("%x\n", out)
}
