import { LATEST_GETH_RELEASE_URL } from '../constants';

export const fetchLatestReleaseVersionAndName = () => {
  return fetch(LATEST_GETH_RELEASE_URL)
    .then(response => response.json())
    .then(release => {
      return {
        versionNumber: release.tag_name as string,
        releaseName: release.name as string
      };
    });
};
