LZ4 - Library Files
================================

The __lib__ directory contains several files, but you don't necessarily need them all.

To integrate fast LZ4 compression/decompression into your program, you basically just need "**lz4.c**" and "**lz4.h**".

For more compression at the cost of compression speed (while preserving decompression speed), use **lz4hc** on top of regular lz4. `lz4hc` only provides compression functions. It also needs `lz4` to compile properly.

If you want to produce files or data streams compatible with `lz4` command line utility, use **lz4frame**. This library encapsulates lz4-compressed blocks into the [official interoperable frame format]. In order to work properly, lz4frame needs lz4 and lz4hc, and also **xxhash**, which provides error detection algorithm.
(_Advanced stuff_ : It's possible to hide xxhash symbols into a local namespace. This is what `liblz4` does, to avoid symbol duplication in case a user program would link to several libraries containing xxhash symbols.)

A more complex "lz4frame_static.h" is also provided, although its usage is not recommended. It contains definitions which are not guaranteed to remain stable within future versions. Use for static linking ***only***.

The other files are not source code. There are :

 - LICENSE : contains the BSD license text
 - Makefile : script to compile or install lz4 library (static or dynamic)
 - liblz4.pc.in : for pkg-config (make install)

[official interoperable frame format]: ../lz4_Frame_format.md
