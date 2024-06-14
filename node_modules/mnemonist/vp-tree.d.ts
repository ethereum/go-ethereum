/**
 * Mnemonist VPTree Typings
 * =========================
 */
type DistanceFunction<T> = (a: T, b: T) => number;
type QueryMatch<T> = {distance: number, item: T};

export default class VPTree<T> {

  // Members
  distance: DistanceFunction<T>;
  size: number;
  D: number;

  // Constructor
  constructor(distance: DistanceFunction<T>, items: Iterable<T>);

  // Methods
  nearestNeighbors(k: number, query: T): Array<QueryMatch<T>>;
  neighbors(radius: number, query: T): Array<QueryMatch<T>>;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    distance: DistanceFunction<I>
  ): VPTree<I>;
}
