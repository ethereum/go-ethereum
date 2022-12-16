export const getReleaseArch = (filename: string) => {
  const arch = filename.includes('alltools') ? filename.split('-')[3] : filename.split('-')[2];

  switch (arch) {
    case '386':
      return '32-bit';
    case 'amd64':
      return '64-bit';
    case 'arm5':
      return 'ARMv5';
    case 'arm6':
      return 'ARMv6';
    case 'arm7':
      return 'ARMv7';
    case 'arm64':
      return 'ARM64';
    case 'mips':
      return 'MIPS32';
    case 'mipsle':
      return 'MIPS32(le)';
    case 'mips64':
      return 'MIPS64';
    case 'mips64le':
      return 'MIPS64(le)';
    default:
      return 'all';
  }
};
