# Importing Keccak C code

The XKCP project contains various implementations of Keccak-related algorithms. These are the steps to select a specific implementation and import the code into our project.

First, generate the source bundles in XKCP:

```
git clone https://github.com/XKCP/XKCP.git
cd XKCP
git checkout 58b20ec

# Edit "Makefile.build".  After all the <fragment> tags, add the following two <target> tags:
<target name="node32" inherits="KeccakSpongeWidth1600 inplace1600bi"/>
<target name="node64" inherits="KeccakSpongeWidth1600 optimized1600ufull"/>

make node32.pack node64.pack
```

The source files we need are now under XKCP's "bin/.pack/npm32/" and "bin/.pack/npm64/".
- Copy those to our repo under "src/libkeccak-32" and "src/libkeccak-64".
- Update our "binding.gyp" to point to the correct ".c" files.
- Run `npm run rebuild`.

## Implementation Choice

Currently, we're using two of XKCP KeccakP[1600] implementations -- the generic 32-bit-optimized one and the generic 64-bit-optimized one.

XKCP has implementations that use CPU-specific instructions (e.g. Intel AVR) and are likely much faster. It might be worth using those.
