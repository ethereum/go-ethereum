import type {IntoInterator} from './types';

export default function includes<T>(
  target: IntoInterator<T>,
  value: T
): boolean;
