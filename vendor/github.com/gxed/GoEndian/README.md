GoEndian
========

A tool to detect byte order for golang.

A sample test code :

```
package main

import (
    "encoding/binary"
    "fmt"
    "github.com/virtao/GoEndian"
)

func main() {
    printEndian()
    useEndian()
}

func printEndian() {
    fmt.Println("Machine byte order : ")
    if endian.IsBigEndian() {
        fmt.Println("Big Endian")
    } else {
        fmt.Println("Little Endian")
    }
}

func useEndian() {
    var iTest int32 = 0x12345678
    var bTest []byte = make([]byte, 4)
    fmt.Println("Int32 to Bytes : ")

    fmt.Println("0x12345678 to current endian : ")
    endian.Endian.PutUint32(bTest, uint32(iTest))
    fmt.Println(bTest)

    fmt.Println("0x12345678 to big endian : ")
    binary.BigEndian.PutUint32(bTest, uint32(iTest))
    fmt.Println(bTest)

    fmt.Println("0x12345678 to little endian : ")
    binary.LittleEndian.PutUint32(bTest, uint32(iTest))
    fmt.Println(bTest)

}
```

The result output:

```
    Machine byte order : 
    Little Endian
    Int32 to Bytes : 
    0x12345678 to current endian : 
    [120 86 52 18]
    0x12345678 to big endian : 
    [18 52 86 120]
    0x12345678 to little endian : 
    [120 86 52 18]
```