import { ALL_GETH_COMMITS_URL } from '../constants';

export const fetchLatestReleaseCommit = (versionNumber: string) => {
  return fetch(`${ALL_GETH_COMMITS_URL}/${versionNumber}`)
    .then(response => response.json())
    .then(commit => commit.sha.slice(0, 8));
};
