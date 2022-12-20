import { ReleaseData } from '../types';

export const compareReleasesFn = (a: ReleaseData, b: ReleaseData) => {
  const aPublished = new Date(a.published);
  const bPublished = new Date(b.published);
  const sameDate = aPublished.toDateString() === bPublished.toDateString();
  const sameCommit = a.commit.label === b.commit.label;

  if (sameDate && !sameCommit) {
    return aPublished > bPublished ? -1 : 1;
  }

  if (sameDate) {
    return a.release.label.length - b.release.label.length;
  }

  return aPublished > bPublished ? -1 : 1;
};
