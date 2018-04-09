# go-msgio - Message IO

This is a simple package that helps read and write length-delimited slices. It's helpful for building wire protocols.

## Usage

### Reading

```go
import "github.com/libp2p/go-msgio"
rdr := ... // some reader from a wire
mrdr := msgio.NewReader(rdr)

for {
  msg, err := mrdr.ReadMsg()
  if err != nil {
    return err
  }

  doSomething(msg)
}
```

### Writing

```go
import "github.com/libp2p/go-msgio"
wtr := genReader()
mwtr := msgio.NewWriter(wtr)

for {
  msg := genMessage()
  err := mwtr.WriteMsg(msg)
  if err != nil {
    return err
  }
}
```

### Duplex

```go
import "github.com/libp2p/go-msgio"
rw := genReadWriter()
mrw := msgio.NewReadWriter(rw)

for {
  msg, err := mrdr.ReadMsg()
  if err != nil {
    return err
  }

  // echo it back :)
  err = mwtr.WriteMsg(msg)
  if err != nil {
    return err
  }
}
```

### Channels

```go
import "github.com/libp2p/go-msgio"
rw := genReadWriter()
rch := msgio.NewReadChannel(rw)
wch := msgio.NewWriteChannel(rw)

for {
  msg, err := <-rch
  if err != nil {
    return err
  }

  // echo it back :)
  wch<- rw
}
```
