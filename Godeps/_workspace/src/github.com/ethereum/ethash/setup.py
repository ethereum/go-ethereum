#!/usr/bin/env python
from distutils.core import setup, Extension
 
pyethash = Extension('pyethash', 
        sources = [
            'src/python/core.c', 
            'src/libethash/util.c', 
            'src/libethash/internal.c',
            'src/libethash/sha3.c'],
        depends = [
            'src/libethash/ethash.h',
            'src/libethash/compiler.h',
            'src/libethash/data_sizes.h',
            'src/libethash/endian.h',
            'src/libethash/ethash.h',
            'src/libethash/fnv.h',
            'src/libethash/internal.h',
            'src/libethash/sha3.h',
            'src/libethash/util.h'
            ],
        extra_compile_args = ["-Isrc/", "-std=gnu99", "-Wall"])
 
setup (
       name = 'pyethash',
       author = "Matthew Wampler-Doty",
       author_email = "matthew.wampler.doty@gmail.com",
       license = 'GPL',
       version = '23',
       url = 'https://github.com/ethereum/ethash',
       download_url = 'https://github.com/ethereum/ethash/tarball/v23',
       description = 'Python wrappers for ethash, the ethereum proof of work hashing function',
       ext_modules = [pyethash],
      )
