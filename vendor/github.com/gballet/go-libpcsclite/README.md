# go-libpcsclite

A golang implementation of the [libpcpsclite](http://github.com/LudovicRousseau/PCSC) client. It connects to the `pcscd` daemon over sockets.

## Purpose

The goal is for major open source projects to distribute a single binary that doesn't depend on `libpcsclite`. It provides an extra function `CheckPCSCDaemon` that will tell the user if `pcscd` is running.

## Example

```golang
func main() {
	client, err := EstablishContext(2)
	if err != nil {
    fmt.Printf("Error establishing context: %v\n", err)
    os.Exit(1)
	}

	_, err = client.ListReaders()
	if err != nil {
    fmt.Printf("Error getting the list of readers: %v\n", err)
    os.Exit(1)
	}

	card, err := client.Connect(client.readerStateDescriptors[0].Name, ShareShared, ProtocolT0|ProtocolT1)
	if err != nil {
    fmt.Printf("Error connecting: %v\n", err)
    os.Exit(1)
	}

	resp, _, err := card.Transmit([]byte{0, 0xa4, 4, 0, 0xA0, 0, 0, 8, 4, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0})

	card.Disconnect(LeaveCard)
}
```

## TODO

  - [x] Finish this README
  - [x] Lock context
  - [ ] implement missing functions

## License

BSD 3-Clause License

Copyright (c) 2019, Guillaume Ballet
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
