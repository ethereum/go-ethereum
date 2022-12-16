import { ALL_GETH_COMMITS_URL } from '../constants';

export const fetchLatestReleaseCommit = (versionNumber: string) => {
  const headers = new Headers({
    // About personal access tokens https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#about-personal-access-tokens
    Authorization: 'Token ' + process.env.GITHUB_TOKEN_READ_ONLY
  });

  return fetch(`${ALL_GETH_COMMITS_URL}/${versionNumber}`, { headers })
    .then(response => response.json())
    .then(commit => commit.sha.slice(0, 8));
};
