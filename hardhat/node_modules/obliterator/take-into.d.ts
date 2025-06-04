import type {IntoInterator} from './types';

// Requires a resolution of https://github.com/microsoft/TypeScript/issues/1213
// export default function takeInto<C<~>, T>(ArrayClass: new <T>(n: number) => C<T>, iterator: Iterator<T>, n: number): C<T>;
export default function takeInto<T>(
  ArrayClass: new <T>(arrayLength: number) => T[],
  iterator: IntoInterator<T>,
  n: number
): T[];
