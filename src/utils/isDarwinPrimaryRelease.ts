import { OS, ReleaseData } from '../types';

export const isDarwinPrimaryRelease = (r: ReleaseData, os: OS, data: ReleaseData[]) =>
  os === 'darwin' &&
  data
    .slice(0, 2) // get latest build to filter on
    .filter((e: ReleaseData) => e.arch === '64-bit')
    .includes(r);
