{
  'variables': {
    'arch': '<!(node -p "process.arch")',
  },
  'target_default': {
    'cflags': [
      '-Wall',
      '-Wextra',
    ],
  },
  'targets': [
    {
      'target_name': 'keccak',
      'type': 'static_library',
      'conditions': [
        ['arch in ("arm64","ppc64","x64")',
          # For known 64-bit architectures, use the implementation optimized for 64-bit CPUs.
          {
            'sources': [
              './src/libkeccak-64/KeccakSpongeWidth1600.c',
              './src/libkeccak-64/KeccakP-1600-opt64.c',
            ],
          },
          # Otherwise, use the implementation optimized for 32-bit CPUs.
          {
            'sources': [
              './src/libkeccak-32/KeccakSpongeWidth1600.c',
              './src/libkeccak-32/KeccakP-1600-inplace32BI.c',
            ],
          },
        ],
      ],
    },
    {
      'target_name': 'addon',
      'dependencies': [
        'keccak',
      ],
      'sources': [
        './src/addon.cc'
      ],
      'include_dirs': [
        '<!@(node -p "require(\'node-addon-api\').include")',
      ],
      'cflags!': [
        '-fno-exceptions',
      ],
      'cflags_cc!': [
        '-fno-exceptions',
      ],
      'defines': [
        'NAPI_VERSION=3',
      ],
      'conditions': [
        ['arch in ("arm64","ppc64","x64")',
          # For known 64-bit architectures, use the implementation optimized for 64-bit CPUs.
          {
            'include_dirs': [ 'src/libkeccak-64' ],
          },
          # Otherwise, use the implementation optimized for 32-bit CPUs.
          {
            'include_dirs': [ 'src/libkeccak-32' ],
          },
        ],
      ],
      'xcode_settings': {
        'GCC_ENABLE_CPP_EXCEPTIONS': 'YES',
        'CLANG_CXX_LIBRARY': 'libc++',
        'MACOSX_DEPLOYMENT_TARGET': '10.7',
      },
      'msvs_settings': {
        'VCCLCompilerTool': {
          'ExceptionHandling': 1,
        },
      },
    },
  ],
}
