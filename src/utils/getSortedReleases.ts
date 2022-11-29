import { ReleaseData } from './../types';
import { compareReleasesFn } from './compareReleasesFn';

export const getSortedReleases = (releases: ReleaseData[], moreReleases: ReleaseData[] = []) => {
  return releases.concat(moreReleases).sort(compareReleasesFn);
};
