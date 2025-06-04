/**
 * Mnemonist DefaultWeakMap Typings
 * ================================
 */
export default class DefaultWeakMap<K extends object, V> {

  // Constructor
  constructor(factory: (key: K) => V);

  // Methods
  clear(): void;
  set(key: K, value: V): this;
  delete(key: K): boolean;
  has(key: K): boolean;
  get(key: K): V;
  peek(key: K): V | undefined;
  inspect(): any;
}
