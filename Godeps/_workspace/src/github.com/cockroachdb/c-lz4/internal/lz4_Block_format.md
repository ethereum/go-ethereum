LZ4 Block Format Description
============================
Last revised: 2015-05-07.
Author : Yann Collet


This specification is intended for developers
willing to produce LZ4-compatible compressed data blocks
using any programming language.

LZ4 is an LZ77-type compressor with a fixed, byte-oriented encoding.
There is no entropy encoder back-end nor framing layer.
The latter is assumed to be handled by other parts of the system (see [LZ4 Frame format]).
This design is assumed to favor simplicity and speed.
It helps later on for optimizations, compactness, and features.

This document describes only the block format,
not how the compressor nor decompressor actually work.
The correctness of the decompressor should not depend
on implementation details of the compressor, and vice versa.

[LZ4 Frame format]: LZ4_Frame_format.md



Compressed block format
-----------------------
An LZ4 compressed block is composed of sequences.
A sequence is a suite of literals (not-compressed bytes),
followed by a match copy.

Each sequence starts with a token.
The token is a one byte value, separated into two 4-bits fields.
Therefore each field ranges from 0 to 15.


The first field uses the 4 high-bits of the token.
It provides the length of literals to follow.

If the field value is 0, then there is no literal.
If it is 15, then we need to add some more bytes to indicate the full length.
Each additional byte then represent a value from 0 to 255,
which is added to the previous value to produce a total length.
When the byte value is 255, another byte is output.
There can be any number of bytes following the token. There is no "size limit".
(Side note : this is why a not-compressible input block is expanded by 0.4%).

Example 1 : A length of 48 will be represented as :

  - 15 : value for the 4-bits High field
  - 33 : (=48-15) remaining length to reach 48

Example 2 : A length of 280 will be represented as :

  - 15  : value for the 4-bits High field
  - 255 : following byte is maxed, since 280-15 >= 255
  - 10  : (=280 - 15 - 255) ) remaining length to reach 280

Example 3 : A length of 15 will be represented as :

  - 15 : value for the 4-bits High field
  - 0  : (=15-15) yes, the zero must be output

Following the token and optional length bytes, are the literals themselves.
They are exactly as numerous as previously decoded (length of literals).
It's possible that there are zero literal.


Following the literals is the match copy operation.

It starts by the offset.
This is a 2 bytes value, in little endian format
(the 1st byte is the "low" byte, the 2nd one is the "high" byte).

The offset represents the position of the match to be copied from.
1 means "current position - 1 byte".
The maximum offset value is 65535, 65536 cannot be coded.
Note that 0 is an invalid value, not used. 

Then we need to extract the match length.
For this, we use the second token field, the low 4-bits.
Value, obviously, ranges from 0 to 15.
However here, 0 means that the copy operation will be minimal.
The minimum length of a match, called minmatch, is 4. 
As a consequence, a 0 value means 4 bytes, and a value of 15 means 19+ bytes.
Similar to literal length, on reaching the highest possible value (15), 
we output additional bytes, one at a time, with values ranging from 0 to 255.
They are added to total to provide the final match length.
A 255 value means there is another byte to read and add.
There is no limit to the number of optional bytes that can be output this way.
(This points towards a maximum achievable compression ratio of about 250).

With the offset and the matchlength,
the decoder can now proceed to copy the data from the already decoded buffer.
On decoding the matchlength, we reach the end of the compressed sequence,
and therefore start another one.


Parsing restrictions
-----------------------
There are specific parsing rules to respect in order to remain compatible
with assumptions made by the decoder :

1. The last 5 bytes are always literals
2. The last match must start at least 12 bytes before end of block.   
   Consequently, a block with less than 13 bytes cannot be compressed.

These rules are in place to ensure that the decoder
will never read beyond the input buffer, nor write beyond the output buffer.

Note that the last sequence is also incomplete,
and stops right after literals.


Additional notes
-----------------------
There is no assumption nor limits to the way the compressor
searches and selects matches within the source data block.
It could be a fast scan, a multi-probe, a full search using BST,
standard hash chains or MMC, well whatever.

Advanced parsing strategies can also be implemented, such as lazy match,
or full optimal parsing.

All these trade-off offer distinctive speed/memory/compression advantages.
Whatever the method used by the compressor, its result will be decodable
by any LZ4 decoder if it follows the format specification described above.
