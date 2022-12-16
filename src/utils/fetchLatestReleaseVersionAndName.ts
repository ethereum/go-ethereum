import { LATEST_GETH_RELEASE_URL } from '../constants';

export const fetchLatestReleaseVersionAndName = () => {
  const headers = new Headers({
    // About personal access tokens https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#about-personal-access-tokens
    Authorization: 'Token ' + process.env.GITHUB_TOKEN_READ_ONLY
  });

  return fetch(LATEST_GETH_RELEASE_URL, { headers })
    .then(response => response.json())
    .then(release => {
      return {
        versionNumber: release.tag_name as string,
        releaseName: release.name as string
      };
    });
};
