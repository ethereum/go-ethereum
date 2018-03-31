# go-multiplex

A super simple stream muxing library implementing [mplex](https://github.com/libp2p/mplex/).

## Usage

```go
mplex := multiplex.NewMultiplex(mysocket)

s, _ := mplex.NewStream()
s.Write([]byte("Hello World!"))
s.Close()

os, _ := mplex.Accept()
// echo back everything received
io.Copy(os, os)
```
