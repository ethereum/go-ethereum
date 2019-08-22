[![GoDoc](https://godoc.org/github.com/steakknife/hamming?status.png)](https://godoc.org/github.com/steakknife/hamming) [![Build Status](https://travis-ci.org/steakknife/hamming.svg?branch=master)](https://travis-ci.org/steakknife/hamming)


# hamming distance calculations in Go

Copyright © 2014, 2015, 2016, 2018 Barry Allard

[MIT license](MIT-LICENSE.txt)

## Performance

```
$ go test -bench=.
BenchmarkCountBitsInt8PopCnt-4      	300000000	         4.30 ns/op
BenchmarkCountBitsInt16PopCnt-4     	300000000	         3.83 ns/op
BenchmarkCountBitsInt32PopCnt-4     	300000000	         3.64 ns/op
BenchmarkCountBitsInt64PopCnt-4     	500000000	         3.60 ns/op
BenchmarkCountBitsIntPopCnt-4       	300000000	         5.72 ns/op
BenchmarkCountBitsUint8PopCnt-4     	1000000000	         2.98 ns/op
BenchmarkCountBitsUint16PopCnt-4    	500000000	         3.23 ns/op
BenchmarkCountBitsUint32PopCnt-4    	500000000	         3.00 ns/op
BenchmarkCountBitsUint64PopCnt-4    	1000000000	         2.94 ns/op
BenchmarkCountBitsUintPopCnt-4      	300000000	         5.04 ns/op
BenchmarkCountBitsBytePopCnt-4      	300000000	         3.99 ns/op
BenchmarkCountBitsRunePopCnt-4      	300000000	         3.83 ns/op
BenchmarkCountBitsInt8-4            	2000000000	         0.74 ns/op
BenchmarkCountBitsInt16-4           	2000000000	         1.54 ns/op
BenchmarkCountBitsInt32-4           	1000000000	         2.63 ns/op
BenchmarkCountBitsInt64-4           	1000000000	         2.56 ns/op
BenchmarkCountBitsInt-4             	200000000	         7.23 ns/op
BenchmarkCountBitsUint16-4          	2000000000	         1.51 ns/op
BenchmarkCountBitsUint32-4          	500000000	         4.00 ns/op
BenchmarkCountBitsUint64-4          	1000000000	         2.64 ns/op
BenchmarkCountBitsUint64Alt-4       	200000000	         7.60 ns/op
BenchmarkCountBitsUint-4            	300000000	         5.48 ns/op
BenchmarkCountBitsUintReference-4   	100000000	        19.2 ns/op
BenchmarkCountBitsByte-4            	2000000000	         0.75 ns/op
BenchmarkCountBitsByteAlt-4         	1000000000	         2.37 ns/op
BenchmarkCountBitsRune-4            	500000000	         2.85 ns/op
PASS
ok  	_/Users/bmf/Projects/hamming	58.305s
$
```

## Usage

```go
import 'github.com/steakknife/hamming'

// ...

// hamming distance between values
hamming.Byte(0xFF, 0x00) // 8
hamming.Byte(0x00, 0x00) // 0

// just count bits in a byte
hamming.CountBitsByte(0xA5), // 4
```

See help in the [docs](https://godoc.org/github.com/steakknife/hamming)

## Get

    go get -u github.com/steakknife/hamming  # master is always stable

## Source

- On the web: https://github.com/steakknife/hamming

- Git: `git clone https://github.com/steakknife/hamming`

## Contact

- [Feedback](mailto:barry.allard@gmail.com)

- [Issues](https://github.com/steakknife/hamming/issues)

## License

[MIT license](MIT-LICENSE.txt)

Copyright © 2014, 2015, 2016 Barry Allard
