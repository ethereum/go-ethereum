import { ReleaseData } from '../types';

export const compareReleasesFn = (a: ReleaseData, b: ReleaseData) => {
  if (new Date(a.published) > new Date(b.published)) {
    return -1;
  }

  if (new Date(a.published) < new Date(b.published)) {
    return 1;
  }

  return 0;
};
