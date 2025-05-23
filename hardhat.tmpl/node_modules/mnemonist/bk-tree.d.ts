/**
 * Mnemonist BKTree Typings
 * =========================
 */
type DistanceFunction<T> = (a: T, b: T) => number;

export default class BKTree<T> {
  
  // Members
  distance: DistanceFunction<T>;
  size: number;

  // Constructor
  constructor(distance: DistanceFunction<T>);

  // Methods
  add(item: T): this;
  search(n: number, query: T): Array<{item: T, distance: number}>;
  toJSON(): object;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}, distance: DistanceFunction<I>): BKTree<I>;
}