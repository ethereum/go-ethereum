import { LAST_COMMIT_BASE_URL } from '../constants';

export const getLastModifiedDate = async (filePath: string) =>
  fetch(`${LAST_COMMIT_BASE_URL}${filePath}/index.md&page=1&per_page=1`)
    .then(res => res.json())
    .then(commits => commits[0].commit.committer.date)
    .catch(_ =>
      fetch(`${LAST_COMMIT_BASE_URL}${filePath}.md&page=1&per_page=1`)
        .then(res => res.json())
        .then(commits => commits[0].commit.committer.date)
    );
