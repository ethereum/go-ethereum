import { CLASSNAME_PREFIX } from '../constants';

const DEFAULT = 'bash';
const TERMINAL = 'terminal';
const JS = ['javascript', 'js', 'jsx', 'ts', 'tsx'];
const SH = ['sh', 'shell'];
const PY = ['python', 'py'];
const SOL = ['solidity', 'sol'];
const LANGS = [JS, SH, PY, SOL];

export const getProgrammingLanguageName = (code: string) => {
  // If `code` argument matches any of the above, return the first value for the language
  for (const lang of LANGS) {
    if (lang.includes(code.toLowerCase())) return lang[0];
  }
  // Check if `code` argument starts with the CLASSNAME_PREFIX
  const hasLanguageNameProperty = code.startsWith(CLASSNAME_PREFIX);
  // If no matched so far, return default code formatting type
  if (!hasLanguageNameProperty) return DEFAULT;
  // `code` starts with the CLASSNAME_PREFIX, so we need to extract the language name
  const newCode = code.replace(CLASSNAME_PREFIX, '').toLowerCase();
  // If declared to be `terminal`, return the DEFAULT
  if (newCode === TERMINAL) return DEFAULT;
  // If `newCode` argument matches any of the above, return the first value for the language
  for (const lang of LANGS) {
    if (lang.includes(newCode.toLowerCase())) return lang[0];
  }
  // If no match from above, simply return the extracted language name
  return newCode;
};
