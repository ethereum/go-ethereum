import type {IntoInterator} from './types';

type PredicateFunction<T> = (item: T) => boolean;

export default function some<T>(
  target: IntoInterator<T>,
  predicate: PredicateFunction<T>
): boolean;
