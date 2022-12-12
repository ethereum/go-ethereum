import { OS, ReleaseData } from '../types';

export const isLinuxPrimaryRelease = (r: ReleaseData, os: OS, data: ReleaseData[]) =>
  os === 'linux' &&
  data
    .slice(0, 12) // get latest build to filter on
    .filter((e: ReleaseData) => e.arch === '64-bit')
    .includes(r);
