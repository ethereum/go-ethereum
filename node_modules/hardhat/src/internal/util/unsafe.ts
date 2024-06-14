/**
 * This function is a typed version of `Object.keys`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
export const unsafeObjectKeys = Object.keys as <T>(
  o: T
) => Array<Extract<keyof T, string>>;

/**
 * This function is a typed version of `Object.entries`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
export function unsafeObjectEntries<T extends object>(o: T) {
  return Object.entries(o) as Array<[keyof T, T[keyof T]]>;
}
