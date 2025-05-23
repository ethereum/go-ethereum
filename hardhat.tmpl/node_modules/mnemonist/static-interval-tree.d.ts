/**
 * Mnemonist StaticIntervalTree Typings
 * =====================================
 */
type StaticIntervalTreeGetter<T> = (item: T) => number;
type StaticIntervalTreeGettersTuple<T> = [StaticIntervalTreeGetter<T>, StaticIntervalTreeGetter<T>];

export default class StaticIntervalTree<T> {

  // Members
  height: number;
  size: number;

  // Constructor
  constructor(intervals: Array<T>, getters?: StaticIntervalTreeGettersTuple<T>);

  // Methods
  intervalsContainingPoint(point: number): Array<T>;
  intervalsOverlappingInterval(interval: T): Array<T>;
  inspect(): any;
  
  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}): StaticIntervalTree<I>;
}