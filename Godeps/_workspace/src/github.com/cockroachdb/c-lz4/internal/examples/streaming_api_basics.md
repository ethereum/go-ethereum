# LZ4 Streaming API Basics
by *Takayuki Matsuoka*
## LZ4 API sets

LZ4 has the following API sets :

 - "Auto Framing" API (lz4frame.h) :
   This is most recommended API for usual application.
   It guarantees interoperability with other LZ4 framing format compliant tools/libraries
   such as LZ4 command line utility, node-lz4, etc.
 - "Block" API : This is recommended for simple purpose.
   It compress single raw memory block to LZ4 memory block and vice versa.
 - "Streaming" API : This is designed for complex thing.
   For example, compress huge stream data in restricted memory environment.

Basically, you should use "Auto Framing" API.
But if you want to write advanced application, it's time to use Block or Streaming APIs.


## What is difference between Block and Streaming API ?

Block API (de)compresses single contiguous memory block.
In other words, LZ4 library find redundancy from single contiguous memory block.
Streaming API does same thing but (de)compress multiple adjacent contiguous memory block.
So LZ4 library could find more redundancy than Block API.

The following figure shows difference between API and block sizes.
In these figures, original data is splitted to 4KiBytes contiguous chunks.

```
Original Data
    +---------------+---------------+----+----+----+
    | 4KiB Chunk A  | 4KiB Chunk B  | C  | D  |... |
    +---------------+---------------+----+----+----+

Example (1) : Block API, 4KiB Block
    +---------------+---------------+----+----+----+
    | 4KiB Chunk A  | 4KiB Chunk B  | C  | D  |... |
    +---------------+---------------+----+----+----+
    | Block #1      | Block #2      | #3 | #4 |... |
    +---------------+---------------+----+----+----+
    
                    (No Dependency)


Example (2) : Block API, 8KiB Block
    +---------------+---------------+----+----+----+
    | 4KiB Chunk A  | 4KiB Chunk B  | C  | D  |... |
    +---------------+---------------+----+----+----+
    |            Block #1           |Block #2 |... |
    +--------------------+----------+-------+-+----+
          ^              |             ^    |
          |              |             |    |
          +--------------+             +----+
          Internal Dependency          Internal Dependency


Example (3) : Streaming API, 4KiB Block
    +---------------+---------------+-----+----+----+
    | 4KiB Chunk A  | 4KiB Chunk B  | C   | D  |... |
    +---------------+---------------+-----+----+----+
    | Block #1      | Block #2      | #3  | #4 |... |
    +---------------+----+----------+-+---+-+--+----+
          ^              |   ^        | ^   |
          |              |   |        | |   |
          +--------------+   +--------+ +---+
          Dependency         Dependency Dependency
```

 - In example (1), there is no dependency.
   All blocks are compressed independently.
 - In example (2), naturally 8KiBytes block has internal dependency.
   But still block #1 and #2 are compressed independently.
 - In example (3), block #2 has dependency to #1,
   also #3 has dependency to #2 and #1, #4 has #3, #2 and #1, and so on.

Here, we can observe difference between example (2) and (3).
In (2), there's no dependency between chunk B and C, but (3) has dependency between B and C.
This dependency improves compression ratio.


## Restriction of Streaming API

For the efficiency, Streaming API doesn't keep mirror copy of dependent (de)compressed memory.
This means users should keep these dependent (de)compressed memory explicitly.
Usually, "Dependent memory" is previous adjacent contiguous memory up to 64KiBytes.
LZ4 will not access further memories.
