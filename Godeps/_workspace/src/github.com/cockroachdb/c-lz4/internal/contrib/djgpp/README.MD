# lz4 for DOS/djgpp
This file details on how to compile lz4.exe, and liblz4.a for use on DOS/djgpp using
Andrew Wu's build-djgpp cross compilers ([GH][0], [Binaries][1]) on OSX, Linux.

## Setup
* Download a djgpp tarball [binaries][1] for your platform.  
* Extract and install it (`tar jxvf djgpp-linux64-gcc492.tar.bz2`).  Note the path.  We'll assume `/home/user/djgpp`.
* Add the `bin` folder to your `PATH`.  In bash, do `export PATH=/home/user/djgpp/bin:$PATH`.
* The `Makefile` in `contrib/djgpp/` sets up `CC`, `AR`, `LD` for you.  So, `CC=i586-pc-msdosdjgpp-gcc`, `AR=i586-pc-msdosdjgpp-ar`, `LD=i586-pc-msdosdjgpp-gcc`.

## Building LZ4 for DOS
In the base dir of lz4 and with `contrib/djgpp/Makefile`, try:
Try:
* `make -f contrib/djgpp/Makefile`
* `make -f contrib/djgpp/Makefile liblz4.a`
* `make -f contrib/djgpp/Makefile lz4.exe`
* `make -f contrib/djgpp/Makefile DESTDIR=/home/user/dos install`, however it doesn't make much sense on a \*nix.
* You can also do `make -f contrib/djgpp/Makefile uninstall`

[0]: https://github.com/andrewwutw/build-djgpp
[1]: https://github.com/andrewwutw/build-djgpp/releases
