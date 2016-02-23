# LZ4 Streaming API Example : Line by Line Text Compression
by *Takayuki Matsuoka*

`blockStreaming_lineByLine.c` is LZ4 Straming API example which implements line by line incremental (de)compression.

Please note the following restrictions :

 - Firstly, read "LZ4 Streaming API Basics".
 - This is relatively advanced application example.
 - Output file is not compatible with lz4frame and platform dependent.


## What's the point of this example ?

 - Line by line incremental (de)compression.
 - Handle huge file in small amount of memory
 - Generally better compression ratio than Block API
 - Non-uniform block size


## How the compression works

First of all, allocate "Ring Buffer" for input and LZ4 compressed data buffer for output.

```
(1)
    Ring Buffer

    +--------+
    | Line#1 |
    +---+----+
        |
        v
     {Out#1}


(2)
    Prefix Mode Dependency
          +----+
          |    |
          v    |
    +--------+-+------+
    | Line#1 | Line#2 |
    +--------+---+----+
                 |
                 v
              {Out#2}


(3)
          Prefix   Prefix
          +----+   +----+
          |    |   |    |
          v    |   v    |
    +--------+-+------+-+------+
    | Line#1 | Line#2 | Line#3 |
    +--------+--------+---+----+
                          |
                          v
                       {Out#3}


(4)
                        External Dictionary Mode
                +----+   +----+
                |    |   |    |
                v    |   v    |
    ------+--------+-+------+-+--------+
          |  ....  | Line#X | Line#X+1 |
    ------+--------+--------+-----+----+
                            ^     |
                            |     v
                            |  {Out#X+1}
                            |
                          Reset


(5)
                                    Prefix
                                    +-----+
                                    |     |
                                    v     |
    ------+--------+--------+----------+--+-------+
          |  ....  | Line#X | Line#X+1 | Line#X+2 |
    ------+--------+--------+----------+-----+----+
                            ^                |
                            |                v
                            |            {Out#X+2}
                            |
                          Reset
```

Next (see (1)), read first line to ringbuffer and compress it by `LZ4_compress_continue()`.
For the first time, LZ4 doesn't know any previous dependencies,
so it just compress the line without dependencies and generates compressed line {Out#1} to LZ4 compressed data buffer.
After that, write {Out#1} to the file and forward ringbuffer offset.

Do the same things to second line (see (2)).
But in this time, LZ4 can use dependency to Line#1 to improve compression ratio.
This dependency is called "Prefix mode".

Eventually, we'll reach end of ringbuffer at Line#X (see (4)).
This time, we should reset ringbuffer offset.
After resetting, at Line#X+1 pointer is not adjacent, but LZ4 still maintain its memory.
This is called "External Dictionary Mode".

In Line#X+2 (see (5)), finally LZ4 forget almost all memories but still remains Line#X+1.
This is the same situation as Line#2.

Continue these procedure to the end of text file.


## How the decompression works

Decompression will do reverse order.

 - Read compressed line from the file to buffer.
 - Decompress it to the ringbuffer.
 - Output decompressed plain text line to the file.
 - Forward ringbuffer offset. If offset exceedes end of the ringbuffer, reset it.

Continue these procedure to the end of the compressed file.
