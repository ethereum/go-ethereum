export const getReleaseCommitHash = (filename: string) => {
  return filename.split('-').reverse()[0].split('.')[0];
};
