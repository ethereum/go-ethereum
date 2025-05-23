import type {IntoInterator} from './types';

type PredicateFunction<T> = (item: T) => boolean;

export default function find<T>(
  target: IntoInterator<T>,
  predicate: PredicateFunction<T>
): T | undefined;
