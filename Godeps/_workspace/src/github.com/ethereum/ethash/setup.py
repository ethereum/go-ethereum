#!/usr/bin/env python
from distutils.core import setup, Extension
 
pyethash_core = Extension('pyethash.core', 
        sources = [
            'src/python/core.c', 
            'src/libethash/util.c', 
            'src/libethash/internal.c',
            'src/libethash/sha3.c'
            ],
        extra_compile_args = ["-std=gnu99"])
 
setup (
       name = 'pyethash',
       author = "Matthew Wampler-Doty",
       author_email = "matthew.wampler.doty@gmail.com",
       license = 'GPL',
       version = '1.0',
       description = 'Python wrappers for ethash, the ethereum proof of work hashing function',
       ext_modules = [pyethash_core],
      )
