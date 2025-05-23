/**
 * Mnemonist HashedArrayTree Typings
 * ==================================
 */
import {IArrayLikeConstructor} from './utils/types';

type HashedArrayTreeOptions = {
  initialCapacity?: number;
  initialLength?: number;
  blockSize?: number;
}

export default class HashedArrayTree<T> {

  // Members
  blockSize: number;
  capacity: number;
  length: number;

  // Constructor
  constructor(ArrayClass: IArrayLikeConstructor, capacity: number);
  constructor(ArrayClass: IArrayLikeConstructor, options: HashedArrayTreeOptions);

  // Methods
  set(index: number, value: T): this;
  get(index: number): T | undefined;
  grow(capacity: number): this;
  resize(length: number): this;
  push(value: T): number;
  pop(): T | undefined;
  inspect(): any;
}
