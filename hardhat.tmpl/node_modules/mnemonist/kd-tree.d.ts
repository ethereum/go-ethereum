/**
 * Mnemonist KDTree Typings
 * =========================
 */
import {IArrayLike} from './utils/types';

export default class KDTree<V> {

  // Members
  dimensions: number;
  size: number;
  visited: number;

  // Methods
  nearestNeighbor(point: Array<number>): V;
  kNearestNeighbors(k: number, point: Array<number>): Array<V>;
  linearKNearestNeighbors(k: number, point: Array<number>): Array<V>;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<[I, Array<number>]>, dimensions: number): KDTree<I>;
  static fromAxes(axes: IArrayLike): KDTree<number>;
  static fromAxes<I>(axes: IArrayLike, labels: Array<I>): KDTree<I>;
}

