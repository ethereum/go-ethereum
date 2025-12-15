import { ReleaseData } from './../types';
import { compareReleasesFn } from './compareReleasesFn';

export const getSortedReleases = (...releases: ReleaseData[][]) => {
  return releases.flat().sort(compareReleasesFn);
};
