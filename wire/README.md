# ethwire

The ethwire package contains the ethereum wire protocol. The ethwire
package is required to write and read from the ethereum network.

# Installation

`go get github.com/ethereum/ethwire-go`

# Messaging overview

The Ethereum Wire protocol defines the communication between the nodes
running Ethereum. Further reader reading can be done on the
[Wiki](http://wiki.ethereum.org/index.php/Wire_Protocol).

# Reading Messages

```go
// Read and validate the next eth message from the provided connection.
// returns a error message with the details.
msg, err := ethwire.ReadMessage(conn)
if err != nil {
  // Handle error
}
```

# Writing Messages

```go
// Constructs a message which can be interpreted by the eth network.
// Write the inventory to network
err := ethwire.WriteMessage(conn, &Msg{
  Type: ethwire.MsgInvTy,
  Data : []interface{}{...},
})
```
