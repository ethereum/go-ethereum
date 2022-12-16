import { LAST_COMMIT_BASE_URL } from '../constants';

export const getLastModifiedDate = async (filePath: string) => {
  const headers = new Headers({
    // About personal access tokens https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#about-personal-access-tokens
    Authorization: 'Token ' + process.env.GITHUB_TOKEN_READ_ONLY
  });

  return fetch(`${LAST_COMMIT_BASE_URL}${filePath}/index.md&page=1&per_page=1`, { headers })
    .then(res => res.json())
    .then(commits => commits[0].commit.committer.date)
    .catch(_ =>
      fetch(`${LAST_COMMIT_BASE_URL}${filePath}.md&page=1&per_page=1`, { headers })
        .then(res => res.json())
        .then(commits => commits[0].commit.committer.date)
        .catch(console.error)
    );
};
