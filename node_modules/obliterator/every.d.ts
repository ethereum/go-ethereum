import type {IntoInterator} from './types';

type PredicateFunction<T> = (item: T) => boolean;

export default function every<T>(
  target: IntoInterator<T>,
  predicate: PredicateFunction<T>
): boolean;
