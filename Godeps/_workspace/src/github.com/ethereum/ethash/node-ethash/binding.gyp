{
  "targets":
    [{
      "target_name": "ethash",
        "sources": [
            './ethash.cc',
            '../libethash/ethash.h',
            '../libethash/util.c',
            '../libethash/util.h',
            '../libethash/blum_blum_shub.h',
            '../libethash/blum_blum_shub.c',
            '../libethash/sha3.h',
            '../libethash/sha3.c',
            '../libethash/internal.h',
            '../libethash/internal.c'
          ],
        "include_dirs": [
          "../",
          "<!(node -e \"require('nan')\")"
        ],
        "cflags": [
        "-Wall",
        "-Wno-maybe-uninitialized",
        "-Wno-uninitialized",
        "-Wno-unused-function",
        "-Wextra"
          ]
    }]
}
