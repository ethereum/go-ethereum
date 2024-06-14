import type {IntoInterator} from './types';

export default function chain<T>(
  ...iterables: IntoInterator<T>[]
): IterableIterator<T>;
