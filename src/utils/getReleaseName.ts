export const getReleaseName = (filename: string) => {
  return filename.includes('alltools')
    ? `Geth & Tools ${filename.split('-')[4]}`
    : `Geth ${filename.split('-')[3]}`;
};
