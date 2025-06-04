/**
 * Returns the plural form of a word.
 *
 * @param n The number of things to represent. This dictates whether to return
 * the singular or plural form of the word.
 * @param singular The singular form of the word.
 * @param plural An optional plural form of the word. If non is given, the
 * plural form is constructed by appending an "s" to the singular form.
 */
export function pluralize(n: number, singular: string, plural?: string) {
  if (n === 1) {
    return singular;
  }

  if (plural !== undefined) {
    return plural;
  }

  return `${singular}s`;
}

/**
 * Replaces all the instances of [[toReplace]] by [[replacement]] in [[str]].
 */
export function replaceAll(
  str: string,
  toReplace: string,
  replacement: string
) {
  return str.split(toReplace).join(replacement);
}
