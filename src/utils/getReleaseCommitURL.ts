import { RELEASE_COMMIT_BASE_URL } from '../constants';

export const getReleaseCommitURL = (hash: string) => `${RELEASE_COMMIT_BASE_URL}${hash}`;
