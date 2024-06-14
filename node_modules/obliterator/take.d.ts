import type {IntoInterator} from './types';

export default function take<T>(
  iterator: IntoInterator<T>,
  n: number
): Array<T>;
