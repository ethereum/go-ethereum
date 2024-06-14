/**
 * Represents the completion of an asynchronous operation
 */
interface ThenPromise<T> extends Promise<T> {
  /**
   * Attaches callbacks for the resolution and/or rejection of the ThenPromise.
   * @param onfulfilled The callback to execute when the ThenPromise is resolved.
   * @param onrejected The callback to execute when the ThenPromise is rejected.
   * @returns A ThenPromise for the completion of which ever callback is executed.
   */
  then<TResult1 = T, TResult2 = never>(onfulfilled?: ((value: T) => TResult1 | PromiseLike<TResult1>) | undefined | null, onrejected?: ((reason: any) => TResult2 | PromiseLike<TResult2>) | undefined | null): ThenPromise<TResult1 | TResult2>;

  /**
   * Attaches a callback for only the rejection of the ThenPromise.
   * @param onrejected The callback to execute when the ThenPromise is rejected.
   * @returns A ThenPromise for the completion of the callback.
   */
  catch<TResult = never>(onrejected?: ((reason: any) => TResult | PromiseLike<TResult>) | undefined | null): ThenPromise<T | TResult>;

  // Extensions specific to then/promise

  /**
   * Attaches callbacks for the resolution and/or rejection of the ThenPromise, without returning a new promise.
   * @param onfulfilled The callback to execute when the ThenPromise is resolved.
   * @param onrejected The callback to execute when the ThenPromise is rejected.
   */
  done(onfulfilled?: ((value: T) => any) | undefined | null, onrejected?: ((reason: any) => any) | undefined | null): void;


  /**
   * Calls a node.js style callback.  If none is provided, the promise is returned.
   */
	nodeify(callback: void | null): ThenPromise<T>;
	nodeify(callback: (err: Error, value: T) => void): void;
}

interface PromiseFulfilledResult<T> {
  status: "fulfilled";
  value: T;
}

interface PromiseRejectedResult {
  status: "rejected";
  reason: any;
}

type PromiseSettledResult<T> = PromiseFulfilledResult<T> | PromiseRejectedResult;

interface ThenPromiseConstructor {
  /**
   * A reference to the prototype.
   */
  readonly prototype: ThenPromise<any>;

  /**
   * Creates a new ThenPromise.
   * @param executor A callback used to initialize the promise. This callback is passed two arguments:
   * a resolve callback used resolve the promise with a value or the result of another promise,
   * and a reject callback used to reject the promise with a provided reason or error.
   */
  new <T>(executor: (resolve: (value?: T | PromiseLike<T>) => void, reject: (reason?: any) => void) => any): ThenPromise<T>;


  /**
   * The any function returns a promise that is fulfilled by the first given promise to be fulfilled, or rejected with an AggregateError containing an array of rejection reasons if all of the given promises are rejected. It resolves all elements of the passed iterable to promises as it runs this algorithm.
   * @param values An array or iterable of Promises.
   * @returns A new Promise.
   */
  any<T extends readonly unknown[] | []>(values: T): Promise<Awaited<T[number]>>;

  /**
   * The any function returns a promise that is fulfilled by the first given promise to be fulfilled, or rejected with an AggregateError containing an array of rejection reasons if all of the given promises are rejected. It resolves all elements of the passed iterable to promises as it runs this algorithm.
   * @param values An array or iterable of Promises.
   * @returns A new Promise.
   */
  any<T>(values: Iterable<T | PromiseLike<T>>): Promise<Awaited<T>>

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5, T6, T7, T8, T9, T10>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>, T10 | PromiseLike<T10>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>, PromiseSettledResult<T6>, PromiseSettledResult<T7>, PromiseSettledResult<T8>, PromiseSettledResult<T9>, PromiseSettledResult<T10>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5, T6, T7, T8, T9>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>, PromiseSettledResult<T6>, PromiseSettledResult<T7>, PromiseSettledResult<T8>, PromiseSettledResult<T9>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5, T6, T7, T8>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>, PromiseSettledResult<T6>, PromiseSettledResult<T7>, PromiseSettledResult<T8>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5, T6, T7>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>, PromiseSettledResult<T6>, PromiseSettledResult<T7>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5, T6>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>, PromiseSettledResult<T6>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4, T5>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>, PromiseSettledResult<T5>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3, T4>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>, PromiseSettledResult<T4>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2, T3>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>, PromiseSettledResult<T3>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T1, T2>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>]): ThenPromise<[PromiseSettledResult<T1>, PromiseSettledResult<T2>]>;

  /**
   * Creates a Promise that is resolved with an array of results when all
   * of the provided Promises resolve or reject.
   * @param values An array of Promises.
   * @returns A new Promise.
   */
  allSettled<T>(values: (T | PromiseLike<T>)[]): ThenPromise<PromiseSettledResult<T>[]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5, T6, T7, T8, T9, T10>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>, T10 | PromiseLike<T10>]): ThenPromise<[T1, T2, T3, T4, T5, T6, T7, T8, T9, T10]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5, T6, T7, T8, T9>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>]): ThenPromise<[T1, T2, T3, T4, T5, T6, T7, T8, T9]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5, T6, T7, T8>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>]): ThenPromise<[T1, T2, T3, T4, T5, T6, T7, T8]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5, T6, T7>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>]): ThenPromise<[T1, T2, T3, T4, T5, T6, T7]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5, T6>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>]): ThenPromise<[T1, T2, T3, T4, T5, T6]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4, T5>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>, T5 | PromiseLike<T5>]): ThenPromise<[T1, T2, T3, T4, T5]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3, T4>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike <T4>]): ThenPromise<[T1, T2, T3, T4]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2, T3>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>]): ThenPromise<[T1, T2, T3]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T1, T2>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>]): ThenPromise<[T1, T2]>;

  /**
   * Creates a ThenPromise that is resolved with an array of results when all of the provided Promises
   * resolve, or rejected when any ThenPromise is rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  all<T>(values: (T | PromiseLike<T>)[]): ThenPromise<T[]>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5, T6, T7, T8, T9, T10>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>, T10 | PromiseLike<T10>]): ThenPromise<T1 | T2 | T3 | T4 | T5 | T6 | T7 | T8 | T9 | T10>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5, T6, T7, T8, T9>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>, T9 | PromiseLike<T9>]): ThenPromise<T1 | T2 | T3 | T4 | T5 | T6 | T7 | T8 | T9>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5, T6, T7, T8>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>, T8 | PromiseLike<T8>]): ThenPromise<T1 | T2 | T3 | T4 | T5 | T6 | T7 | T8>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5, T6, T7>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>, T7 | PromiseLike<T7>]): ThenPromise<T1 | T2 | T3 | T4 | T5 | T6 | T7>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5, T6>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>, T6 | PromiseLike<T6>]): ThenPromise<T1 | T2 | T3 | T4 | T5 | T6>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4, T5>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>, T5 | PromiseLike<T5>]): ThenPromise<T1 | T2 | T3 | T4 | T5>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3, T4>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>, T4 | PromiseLike<T4>]): ThenPromise<T1 | T2 | T3 | T4>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2, T3>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>, T3 | PromiseLike<T3>]): ThenPromise<T1 | T2 | T3>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T1, T2>(values: [T1 | PromiseLike<T1>, T2 | PromiseLike<T2>]): ThenPromise<T1 | T2>;

  /**
   * Creates a ThenPromise that is resolved or rejected when any of the provided Promises are resolved
   * or rejected.
   * @param values An array of Promises.
   * @returns A new ThenPromise.
   */
  race<T>(values: (T | PromiseLike<T>)[]): ThenPromise<T>;

  /**
   * Creates a new rejected promise for the provided reason.
   * @param reason The reason the promise was rejected.
   * @returns A new rejected ThenPromise.
   */
  reject(reason: any): ThenPromise<never>;

  /**
   * Creates a new rejected promise for the provided reason.
   * @param reason The reason the promise was rejected.
   * @returns A new rejected ThenPromise.
   */
  reject<T>(reason: any): ThenPromise<T>;

  /**
   * Creates a new resolved promise for the provided value.
   * @param value A promise.
   * @returns A promise whose internal state matches the provided promise.
   */
  resolve<T>(value: T | PromiseLike<T>): ThenPromise<T>;

  /**
   * Creates a new resolved promise .
   * @returns A resolved promise.
   */
  resolve(): ThenPromise<void>;

  // Extensions specific to then/promise

  denodeify: (fn: Function) => (...args: any[]) => ThenPromise<any>;
  nodeify: (fn: Function) => Function;
}

declare var ThenPromise: ThenPromiseConstructor;

export = ThenPromise;
