import {
  LINUX_BINARY_BASE_URL,
  MACOS_BINARY_BASE_URL,
  WINDOWS_BINARY_BASE_URL
} from '../constants';

export const getLatestBinaryURL = (os: string, versionNumber: string, commit: string) => {
  if (os === 'linux') return `${LINUX_BINARY_BASE_URL}${versionNumber.slice(1)}-${commit}.tar.gz`;

  if (os === 'darwin') return `${MACOS_BINARY_BASE_URL}${versionNumber.slice(1)}-${commit}.tar.gz`;

  return `${WINDOWS_BINARY_BASE_URL}${versionNumber.slice(1)}-${commit}.exe`;
};
