export const getKebabCaseFromName = (name: string): string =>
  name
    .replace(/[#]/g, '')
    .trim()
    .toLowerCase()
    .replace(/ /g, '-')
    .replace(/[^a-z0-9-]/g, '');
