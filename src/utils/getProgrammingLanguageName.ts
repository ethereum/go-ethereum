const CLASSNAME_PREFIX = 'language-';
const DEFAULT = 'bash';
const JS = ['javascript', 'js', 'jsx', 'ts', 'tsx'];
const SH = ['sh', 'shell'];
const PY = ['python', 'py'];
const SOL = ['solidity', 'sol'];
const LANGS = [JS, SH, PY, SOL];

export const getProgrammingLanguageName = (code: string) => {
  for (const lang of LANGS) {
    if (lang.includes(code.toLowerCase())) return lang[0];
  }
  const hasLanguageNameProperty = code.startsWith(CLASSNAME_PREFIX);
  if (!hasLanguageNameProperty) return DEFAULT;
  const newCode = code.replace(CLASSNAME_PREFIX, '').toLowerCase();
  for (const lang of LANGS) {
    if (lang.includes(code.toLowerCase())) return lang[0];
  }
  return newCode;
};
