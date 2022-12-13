import { OS, ReleaseData } from '../types';

export const isMobilePrimaryRelease = (r: ReleaseData, os: OS, data: ReleaseData[]) =>
  os === 'mobile' &&
  data
    .filter((e: ReleaseData) => e.arch === 'all')
    .slice(0, 1) // get latest build
    .includes(r);
