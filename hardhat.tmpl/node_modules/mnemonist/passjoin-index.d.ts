/**
 * Mnemonist PassjoinIndex Typings
 * ================================
 */
type LevenshteinDistanceFunction<T> = (a: T, b: T) => number;

export default class PassjoinIndex<T> implements Iterable<T> {

  // Members
  size: number;

  // Constructor
  constructor(levenshtein: LevenshteinDistanceFunction<T>, k: number);

  // Methods
  add(value: T): this;
  search(query: T): Set<T>;
  clear(): void;
  forEach(callback: (value: T, index: number, self: this) => void, scope?: any): void;
  values(): IterableIterator<T>;
  [Symbol.iterator](): IterableIterator<T>;
  inspect(): any;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    levenshtein: LevenshteinDistanceFunction<I>,
    k: number
  ): PassjoinIndex<I>;
}

export function countKeys(k: number, s: number): number;
export function comparator<T>(a: T, b: T): number;
export function partition(k: number, l: number): Array<[number, number]>;
export function segments<T>(k: number, string: T): Array<T>;
export function segmentPos<T>(k: number, i: number, string: T): number;

export function multiMatchAwareInterval(
  k: number,
  delta: number,
  i: number,
  s: number,
  pi: number,
  li: number
): [number, number];

export function multiMatchAwareSubstrings<T>(
  k: number,
  string: T,
  l: number,
  i: number,
  pi: number,
  li: number
): Array<T>;
