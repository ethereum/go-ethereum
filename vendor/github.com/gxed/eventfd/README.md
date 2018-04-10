# Eventfd
A wrapper around [eventfd()](http://linux.die.net/man/2/eventfd).

## Installation
Download and install eventfd
~~~
go get github.com/sahne/eventfd
~~~

## Usage
~~~ go
package main

import (
	"log"
	"github.com/sahne/eventfd"
)

func main() {
	efd, err := eventfd.New()
	if err != nil {
		log.Fatalf("Could not create EventFD: %v", err)
	}
	/* TODO: register fd at kernel interface (for example cgroups memory watcher) */
	/* listen for new events */
	for {
		val, err := efd.ReadEvents()
		if err != nil {
			log.Printf("Error while reading from eventfd: %v", err)
			break
		}
	}
}
~~~
