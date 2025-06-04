import { Applicative, Applicative1, Applicative2, Applicative2C, Applicative3, Applicative3C } from './Applicative';
import { Separated, Compactable1 } from './Compactable';
import { Either } from './Either';
import { Foldable, Foldable1, Foldable2, Foldable3 } from './Foldable';
import { Predicate, Refinement } from './function';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Magma } from './Magma';
import { Monoid } from './Monoid';
import { Option } from './Option';
import { Semigroup } from './Semigroup';
import { Eq } from './Eq';
import { Unfoldable, Unfoldable1 } from './Unfoldable';
import { Show } from './Show';
import { FunctorWithIndex1 } from './FunctorWithIndex';
import { Foldable2v1 } from './Foldable2v';
import { TraversableWithIndex1 } from './TraversableWithIndex';
import { FilterableWithIndex1, PredicateWithIndex, RefinementWithIndex } from './FilterableWithIndex';
import { Witherable1 } from './Witherable';
import { FoldableWithIndex1 } from './FoldableWithIndex';
/**
 * @since 1.17.0
 */
export declare const getShow: <A>(S: Show<A>) => Show<Record<string, A>>;
/**
 * Calculate the number of key/value pairs in a record
 *
 * @since 1.10.0
 */
export declare const size: <A>(d: Record<string, A>) => number;
/**
 * Test whether a record is empty
 *
 * @since 1.10.0
 */
export declare const isEmpty: <A>(d: Record<string, A>) => boolean;
/**
 * Map a record into an array
 *
 * @example
 * import {collect} from 'fp-ts/lib/Record'
 *
 * const ob: {a: string, b: boolean} = {a: 'foo', b: false}
 * assert.deepStrictEqual(
 *   collect(ob, (key, val) => ({key: key, value: val})),
 *   [{key: 'a', value: 'foo'}, {key: 'b', value: false}]
 * )
 *
 * @since 1.10.0
 */
export declare function collect<K extends string, A, B>(f: (k: K, a: A) => B): (d: Record<K, A>) => Array<B>;
export declare function collect<A, B>(f: (k: string, a: A) => B): (d: Record<string, A>) => Array<B>;
/** @deprecated */
export declare function collect<K extends string, A, B>(d: Record<K, A>, f: (k: K, a: A) => B): Array<B>;
/** @deprecated */
export declare function collect<A, B>(d: Record<string, A>, f: (k: string, a: A) => B): Array<B>;
/**
 * @since 1.10.0
 */
export declare function toArray<K extends string, A>(d: Record<K, A>): Array<[K, A]>;
export declare function toArray<A>(d: Record<string, A>): Array<[string, A]>;
/**
 * Unfolds a record into a list of key/value pairs
 *
 * @since 1.10.0
 */
export declare function toUnfoldable<F extends URIS>(unfoldable: Unfoldable1<F>): <K extends string, A>(d: Record<K, A>) => Kind<F, [K, A]>;
export declare function toUnfoldable<F>(unfoldable: Unfoldable<F>): <K extends string, A>(d: Record<K, A>) => HKT<F, [K, A]>;
/**
 * Use `insertAt`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function insert<KS extends string, K extends string, A>(k: K, a: A, d: Record<KS, A>): Record<KS | K, A>;
/** @deprecated */
export declare function insert<A>(k: string, a: A, d: Record<string, A>): Record<string, A>;
/**
 * Use `deleteAt`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function remove<KS extends string, K extends string, A>(k: K, d: Record<KS, A>): Record<string extends K ? string : Exclude<KS, K>, A>;
/** @deprecated */
export declare function remove<A>(k: string, d: Record<string, A>): Record<string, A>;
/**
 * Delete a key and value from a map, returning the value as well as the subsequent map
 *
 * @since 1.10.0
 */
export declare function pop<A>(k: string): (d: Record<string, A>) => Option<[A, Record<string, A>]>;
/** @deprecated */
export declare function pop<A>(k: string, d: Record<string, A>): Option<[A, Record<string, A>]>;
/**
 * Test whether one record contains all of the keys and values contained in another record
 *
 * @since 1.14.0
 */
export declare const isSubrecord: <A>(E: Eq<A>) => (d1: Record<string, A>, d2: Record<string, A>) => boolean;
/**
 * Use `isSubrecord` instead
 * @since 1.10.0
 * @deprecated
 */
export declare const isSubdictionary: <A>(E: Eq<A>) => (d1: Record<string, A>, d2: Record<string, A>) => boolean;
/**
 * Use `getEq`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare const getSetoid: typeof getEq;
/**
 * @since 1.19.0
 */
export declare function getEq<K extends string, A>(E: Eq<A>): Eq<Record<K, A>>;
export declare function getEq<A>(E: Eq<A>): Eq<Record<string, A>>;
/**
 * Returns a `Semigroup` instance for records given a `Semigroup` instance for their values
 *
 * @example
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 * import { getMonoid } from 'fp-ts/lib/Record'
 *
 * const M = getMonoid(semigroupSum)
 * assert.deepStrictEqual(M.concat({ foo: 123 }, { foo: 456 }), { foo: 579 })
 *
 * @since 1.10.0
 */
export declare function getMonoid<K extends string, A>(S: Semigroup<A>): Monoid<Record<K, A>>;
export declare function getMonoid<A>(S: Semigroup<A>): Monoid<Record<string, A>>;
/**
 * Lookup the value for a key in a record
 * @since 1.10.0
 */
export declare const lookup: <A>(key: string, fa: Record<string, A>) => Option<A>;
/**
 * @since 1.10.0
 */
export declare function filter<A, B extends A>(refinement: Refinement<A, B>): (fa: Record<string, A>) => Record<string, B>;
export declare function filter<A>(predicate: Predicate<A>): (fa: Record<string, A>) => Record<string, A>;
/** @deprecated */
export declare function filter<A, B extends A>(fa: Record<string, A>, refinement: Refinement<A, B>): Record<string, B>;
/** @deprecated */
export declare function filter<A>(fa: Record<string, A>, predicate: Predicate<A>): Record<string, A>;
/**
 * @since 1.10.0
 */
export declare const empty: Record<string, never>;
/**
 * Use `mapWithIndex`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function mapWithKey<K extends string, A, B>(fa: Record<K, A>, f: (k: K, a: A) => B): Record<K, B>;
export declare function mapWithKey<A, B>(fa: Record<string, A>, f: (k: string, a: A) => B): Record<string, B>;
/**
 * Map a record passing the values to the iterating function
 * @since 1.10.0
 */
export declare function map<A, B>(f: (a: A) => B): <K extends string>(fa: Record<K, A>) => Record<K, B>;
/** @deprecated */
export declare function map<K extends string, A, B>(fa: Record<K, A>, f: (a: A) => B): Record<K, B>;
/** @deprecated */
export declare function map<A, B>(fa: Record<string, A>, f: (a: A) => B): Record<string, B>;
/**
 * Reduce object by iterating over it's values.
 *
 * @since 1.10.0
 *
 * @example
 * import { reduce } from 'fp-ts/lib/Record'
 *
 * const joinAllVals = (ob: {[k: string]: string}) => reduce(ob, '', (acc, val) => acc + val)
 *
 * assert.deepStrictEqual(joinAllVals({a: 'foo', b: 'bar'}), 'foobar')
 */
export declare function reduce<A, B>(fa: Record<string, A>, b: B, f: (b: B, a: A) => B): B;
/**
 * @since 1.10.0
 */
export declare function foldMap<M>(M: Monoid<M>): {
    <A>(f: (a: A) => M): (fa: Record<string, A>) => M;
    /** @deprecated */
    <A>(fa: Record<string, A>, f: (a: A) => M): M;
};
/**
 * Use `reduceRight`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function foldr<A, B>(fa: Record<string, A>, b: B, f: (a: A, b: B) => B): B;
/**
 * Use `reduceWithIndex`
 *
 * @since 1.12.0
 * @deprecated
 */
export declare function reduceWithKey<K extends string, A, B>(fa: Record<K, A>, b: B, f: (k: K, b: B, a: A) => B): B;
export declare function reduceWithKey<A, B>(fa: Record<string, A>, b: B, f: (k: string, b: B, a: A) => B): B;
/**
 * Use `foldMapWithIndex`
 *
 * @since 1.12.0
 * @deprecated
 */
export declare const foldMapWithKey: <M>(M: Monoid<M>) => <A>(fa: Record<string, A>, f: (k: string, a: A) => M) => M;
/**
 * Use `reduceRightWithIndex`
 *
 * @since 1.12.0
 * @deprecated
 */
export declare function foldrWithKey<K extends string, A, B>(fa: Record<K, A>, b: B, f: (k: K, a: A, b: B) => B): B;
export declare function foldrWithKey<A, B>(fa: Record<string, A>, b: B, f: (k: string, a: A, b: B) => B): B;
/**
 * Create a record with one key/value pair
 *
 * @since 1.10.0
 */
export declare const singleton: <K extends string, A>(k: K, a: A) => Record<K, A>;
/**
 * Use `traverseWithIndex`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function traverseWithKey<F extends URIS3>(F: Applicative3<F>): <U, L, A, B>(ta: Record<string, A>, f: (k: string, a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Record<string, B>>;
/** @deprecated */
export declare function traverseWithKey<F extends URIS2>(F: Applicative2<F>): <L, A, B>(ta: Record<string, A>, f: (k: string, a: A) => Kind2<F, L, B>) => Kind2<F, L, Record<string, B>>;
/** @deprecated */
export declare function traverseWithKey<F extends URIS>(F: Applicative1<F>): <A, B>(ta: Record<string, A>, f: (k: string, a: A) => Kind<F, B>) => Kind<F, Record<string, B>>;
/** @deprecated */
export declare function traverseWithKey<F>(F: Applicative<F>): <A, B>(ta: Record<string, A>, f: (k: string, a: A) => HKT<F, B>) => HKT<F, Record<string, B>>;
/**
 * Use `traverse2v`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function traverse<F extends URIS3>(F: Applicative3<F>): <U, L, A, B>(ta: Record<string, A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Record<string, B>>;
/** @deprecated */
export declare function traverse<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A, B>(ta: Record<string, A>, f: (a: A) => Kind3<F, U, L, B>) => Kind3<F, U, L, Record<string, B>>;
/** @deprecated */
export declare function traverse<F extends URIS2>(F: Applicative2<F>): <L, A, B>(ta: Record<string, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Record<string, B>>;
/** @deprecated */
export declare function traverse<F extends URIS2, L>(F: Applicative2C<F, L>): <A, B>(ta: Record<string, A>, f: (a: A) => Kind2<F, L, B>) => Kind2<F, L, Record<string, B>>;
/** @deprecated */
export declare function traverse<F extends URIS>(F: Applicative1<F>): <A, B>(ta: Record<string, A>, f: (a: A) => Kind<F, B>) => Kind<F, Record<string, B>>;
/** @deprecated */
export declare function traverse<F>(F: Applicative<F>): <A, B>(ta: Record<string, A>, f: (a: A) => HKT<F, B>) => HKT<F, Record<string, B>>;
/**
 * @since 1.10.0
 */
export declare function sequence<F extends URIS3>(F: Applicative3<F>): <U, L, A>(ta: Record<string, Kind3<F, U, L, A>>) => Kind3<F, U, L, Record<string, A>>;
export declare function sequence<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A>(ta: Record<string, Kind3<F, U, L, A>>) => Kind3<F, U, L, Record<string, A>>;
export declare function sequence<F extends URIS2>(F: Applicative2<F>): <L, A>(ta: Record<string, Kind2<F, L, A>>) => Kind2<F, L, Record<string, A>>;
export declare function sequence<F extends URIS2, L>(F: Applicative2C<F, L>): <A>(ta: Record<string, Kind2<F, L, A>>) => Kind2<F, L, Record<string, A>>;
export declare function sequence<F extends URIS>(F: Applicative1<F>): <A>(ta: Record<string, Kind<F, A>>) => Kind<F, Record<string, A>>;
export declare function sequence<F>(F: Applicative<F>): <A>(ta: Record<string, HKT<F, A>>) => HKT<F, Record<string, A>>;
/**
 * @since 1.10.0
 */
export declare const compact: <A>(fa: Record<string, Option<A>>) => Record<string, A>;
/**
 * @since 1.10.0
 */
export declare function partitionMap<RL, RR, A>(f: (a: A) => Either<RL, RR>): (fa: Record<string, A>) => Separated<Record<string, RL>, Record<string, RR>>;
/** @deprecated */
export declare function partitionMap<RL, RR, A>(fa: Record<string, A>, f: (a: A) => Either<RL, RR>): Separated<Record<string, RL>, Record<string, RR>>;
/**
 * @since 1.10.0
 */
export declare function partition<A>(predicate: Predicate<A>): (fa: Record<string, A>) => Separated<Record<string, A>, Record<string, A>>;
/** @deprecated */
export declare function partition<A>(fa: Record<string, A>, predicate: Predicate<A>): Separated<Record<string, A>, Record<string, A>>;
/**
 * @since 1.10.0
 */
export declare function separate<RL, RR>(fa: Record<string, Either<RL, RR>>): Separated<Record<string, RL>, Record<string, RR>>;
/**
 * Use `record.wither`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function wither<F extends URIS3>(F: Applicative3<F>): <U, L, A, B>(wa: Record<string, A>, f: (a: A) => Kind3<F, U, L, Option<B>>) => Kind3<F, U, L, Record<string, B>>;
export declare function wither<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <A, B>(wa: Record<string, A>, f: (a: A) => Kind3<F, U, L, Option<B>>) => Kind3<F, U, L, Record<string, B>>;
export declare function wither<F extends URIS2>(F: Applicative2<F>): <L, A, B>(wa: Record<string, A>, f: (a: A) => Kind2<F, L, Option<B>>) => Kind2<F, L, Record<string, B>>;
export declare function wither<F extends URIS2, L>(F: Applicative2C<F, L>): <A, B>(wa: Record<string, A>, f: (a: A) => Kind2<F, L, Option<B>>) => Kind2<F, L, Record<string, B>>;
export declare function wither<F extends URIS>(F: Applicative1<F>): <A, B>(wa: Record<string, A>, f: (a: A) => Kind<F, Option<B>>) => Kind<F, Record<string, B>>;
export declare function wither<F>(F: Applicative<F>): <A, B>(wa: Record<string, A>, f: (a: A) => HKT<F, Option<B>>) => HKT<F, Record<string, B>>;
/**
 * Use `record.wilt`
 *
 * @since 1.10.0
 * @deprecated
 */
export declare function wilt<F extends URIS3>(F: Applicative3<F>): <U, L, RL, RR, A>(wa: Record<string, A>, f: (a: A) => Kind3<F, U, L, Either<RL, RR>>) => Kind3<F, U, L, Separated<Record<string, RL>, Record<string, RR>>>;
export declare function wilt<F extends URIS3, U, L>(F: Applicative3C<F, U, L>): <RL, RR, A>(wa: Record<string, A>, f: (a: A) => Kind3<F, U, L, Either<RL, RR>>) => Kind3<F, U, L, Separated<Record<string, RL>, Record<string, RR>>>;
export declare function wilt<F extends URIS2>(F: Applicative2<F>): <L, RL, RR, A>(wa: Record<string, A>, f: (a: A) => Kind2<F, L, Either<RL, RR>>) => Kind2<F, L, Separated<Record<string, RL>, Record<string, RR>>>;
export declare function wilt<F extends URIS2, L>(F: Applicative2C<F, L>): <RL, RR, A>(wa: Record<string, A>, f: (a: A) => Kind2<F, L, Either<RL, RR>>) => Kind2<F, L, Separated<Record<string, RL>, Record<string, RR>>>;
export declare function wilt<F extends URIS>(F: Applicative1<F>): <RL, RR, A>(wa: Record<string, A>, f: (a: A) => Kind<F, Either<RL, RR>>) => Kind<F, Separated<Record<string, RL>, Record<string, RR>>>;
export declare function wilt<F>(F: Applicative<F>): <RL, RR, A>(wa: Record<string, A>, f: (a: A) => HKT<F, Either<RL, RR>>) => HKT<F, Separated<Record<string, RL>, Record<string, RR>>>;
/**
 * @since 1.10.0
 */
export declare function filterMap<A, B>(f: (a: A) => Option<B>): (fa: Record<string, A>) => Record<string, B>;
/** @deprecated */
export declare function filterMap<A, B>(fa: Record<string, A>, f: (a: A) => Option<B>): Record<string, B>;
/**
 * Use `partitionMapWithIndex`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare function partitionMapWithKey<K extends string, RL, RR, A>(fa: Record<K, A>, f: (key: K, a: A) => Either<RL, RR>): Separated<Record<string, RL>, Record<string, RR>>;
export declare function partitionMapWithKey<RL, RR, A>(fa: Record<string, A>, f: (key: string, a: A) => Either<RL, RR>): Separated<Record<string, RL>, Record<string, RR>>;
/**
 * Use `partitionWithIndex`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare function partitionWithKey<K extends string, A>(fa: Record<K, A>, predicate: (key: K, a: A) => boolean): Separated<Record<string, A>, Record<string, A>>;
export declare function partitionWithKey<A>(fa: Record<string, A>, predicate: (key: string, a: A) => boolean): Separated<Record<string, A>, Record<string, A>>;
/**
 * Use `filterMapWithIndex`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare function filterMapWithKey<K extends string, A, B>(fa: Record<K, A>, f: (key: K, a: A) => Option<B>): Record<string, B>;
export declare function filterMapWithKey<A, B>(fa: Record<string, A>, f: (key: string, a: A) => Option<B>): Record<string, B>;
/**
 * Use `filterWithIndex`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare function filterWithKey<K extends string, A>(fa: Record<K, A>, predicate: (key: K, a: A) => boolean): Record<string, A>;
export declare function filterWithKey<A>(fa: Record<string, A>, predicate: (key: string, a: A) => boolean): Record<string, A>;
/**
 * Create a record from a foldable collection of key/value pairs, using the
 * specified function to combine values for duplicate keys.
 *
 * @since 1.10.0
 */
export declare function fromFoldable<F extends URIS3>(F: Foldable3<F>): <K extends string, U, L, A>(ta: Kind3<F, U, L, [K, A]>, onConflict: (existing: A, a: A) => A) => Record<K, A>;
export declare function fromFoldable<F extends URIS2>(F: Foldable2<F>): <K extends string, L, A>(ta: Kind2<F, L, [K, A]>, onConflict: (existing: A, a: A) => A) => Record<K, A>;
export declare function fromFoldable<F extends URIS>(F: Foldable1<F>): <K extends string, A>(ta: Kind<F, [K, A]>, onConflict: (existing: A, a: A) => A) => Record<K, A>;
export declare function fromFoldable<F>(F: Foldable<F>): <K extends string, A>(ta: HKT<F, [K, A]>, onConflict: (existing: A, a: A) => A) => Record<K, A>;
/**
 * Create a record from a foldable collection using the specified functions to
 *
 * - map to key/value pairs
 * - combine values for duplicate keys.
 *
 * @example
 * import { getLastSemigroup } from 'fp-ts/lib/Semigroup'
 * import { array, zip } from 'fp-ts/lib/Array'
 * import { identity } from 'fp-ts/lib/function'
 * import { fromFoldableMap } from 'fp-ts/lib/Record'
 *
 * // like lodash `zipObject` or ramda `zipObj`
 * export const zipObject = <K extends string, A>(keys: Array<K>, values: Array<A>): Record<K, A> =>
 *   fromFoldableMap(getLastSemigroup<A>(), array)(zip(keys, values), identity)
 *
 * assert.deepStrictEqual(zipObject(['a', 'b'], [1, 2, 3]), { a: 1, b: 2 })
 *
 * // build a record from a field
 * interface User {
 *   id: string
 *   name: string
 * }
 *
 * const users: Array<User> = [
 *   { id: 'id1', name: 'name1' },
 *   { id: 'id2', name: 'name2' },
 *   { id: 'id1', name: 'name3' }
 * ]
 *
 * assert.deepStrictEqual(fromFoldableMap(getLastSemigroup<User>(), array)(users, user => [user.id, user]), {
 *   id1: { id: 'id1', name: 'name3' },
 *   id2: { id: 'id2', name: 'name2' }
 * })
 *
 * @since 1.16.0
 */
export declare function fromFoldableMap<F extends URIS3, B>(M: Magma<B>, F: Foldable3<F>): <U, L, A, K extends string>(ta: Kind3<F, U, L, A>, f: (a: A) => [K, B]) => Record<K, B>;
export declare function fromFoldableMap<F extends URIS2, B>(M: Magma<B>, F: Foldable2<F>): <L, A, K extends string>(ta: Kind2<F, L, A>, f: (a: A) => [K, B]) => Record<K, B>;
export declare function fromFoldableMap<F extends URIS, B>(M: Magma<B>, F: Foldable1<F>): <A, K extends string>(ta: Kind<F, A>, f: (a: A) => [K, B]) => Record<K, B>;
export declare function fromFoldableMap<F, B>(M: Magma<B>, F: Foldable<F>): <A, K extends string>(ta: HKT<F, A>, f: (a: A) => [K, B]) => Record<K, B>;
/**
 * @since 1.14.0
 */
export declare function every<A>(fa: {
    [key: string]: A;
}, predicate: (a: A) => boolean): boolean;
/**
 * @since 1.14.0
 */
export declare function some<A>(fa: {
    [key: string]: A;
}, predicate: (a: A) => boolean): boolean;
/**
 * @since 1.14.0
 */
export declare function elem<A>(E: Eq<A>): (a: A, fa: {
    [key: string]: A;
}) => boolean;
/**
 * @since 1.12.0
 */
export declare function partitionMapWithIndex<K extends string, RL, RR, A>(f: (key: K, a: A) => Either<RL, RR>): (fa: Record<K, A>) => Separated<Record<string, RL>, Record<string, RR>>;
/** @deprecated */
export declare function partitionMapWithIndex<K extends string, RL, RR, A>(fa: Record<K, A>, f: (key: K, a: A) => Either<RL, RR>): Separated<Record<string, RL>, Record<string, RR>>;
/** @deprecated */
export declare function partitionMapWithIndex<RL, RR, A>(fa: Record<string, A>, f: (key: string, a: A) => Either<RL, RR>): Separated<Record<string, RL>, Record<string, RR>>;
/**
 * @since 1.12.0
 */
export declare function partitionWithIndex<K extends string, A, B extends A>(refinementWithIndex: RefinementWithIndex<K, A, B>): (fa: Record<K, A>) => Separated<Record<string, A>, Record<string, B>>;
export declare function partitionWithIndex<K extends string, A>(predicateWithIndex: PredicateWithIndex<K, A>): (fa: Record<K, A>) => Separated<Record<string, A>, Record<string, A>>;
/** @deprecated */
export declare function partitionWithIndex<K extends string, A>(fa: Record<K, A>, p: (key: K, a: A) => boolean): Separated<Record<string, A>, Record<string, A>>;
/** @deprecated */
export declare function partitionWithIndex<A>(fa: Record<string, A>, p: (key: string, a: A) => boolean): Separated<Record<string, A>, Record<string, A>>;
/**
 * @since 1.12.0
 */
export declare function filterMapWithIndex<K extends string, A, B>(f: (key: K, a: A) => Option<B>): (fa: Record<K, A>) => Record<string, B>;
/** @deprecated */
export declare function filterMapWithIndex<K extends string, A, B>(fa: Record<K, A>, f: (key: K, a: A) => Option<B>): Record<string, B>;
/** @deprecated */
export declare function filterMapWithIndex<A, B>(fa: Record<string, A>, f: (key: string, a: A) => Option<B>): Record<string, B>;
/**
 * @since 1.12.0
 */
export declare function filterWithIndex<K extends string, A, B extends A>(refinementWithIndex: RefinementWithIndex<K, A, B>): (fa: Record<K, A>) => Record<string, B>;
export declare function filterWithIndex<K extends string, A>(predicateWithIndex: PredicateWithIndex<K, A>): (fa: Record<K, A>) => Record<string, A>;
/** @deprecated */
export declare function filterWithIndex<K extends string, A>(fa: Record<K, A>, p: (key: K, a: A) => boolean): Record<string, A>;
/** @deprecated */
export declare function filterWithIndex<A>(fa: Record<string, A>, p: (key: string, a: A) => boolean): Record<string, A>;
/**
 * Insert or replace a key/value pair in a map
 *
 * @since 1.19.0
 */
export declare function insertAt<K extends string, A>(k: K, a: A): <KS extends string>(r: Record<KS, A>) => Record<KS | K, A>;
/**
 * Delete a key and value from a map
 *
 * @since 1.19.0
 */
export declare function deleteAt<K extends string>(k: K): <KS extends string, A>(d: Record<KS, A>) => Record<string extends K ? string : Exclude<KS, K>, A>;
declare module './HKT' {
    interface URItoKind<A> {
        Record: Record<string, A>;
    }
}
/**
 * @since 1.19.0
 */
export declare const URI = "Record";
/**
 * @since 1.19.0
 */
export declare type URI = typeof URI;
/**
 * Map a record passing the keys to the iterating function
 *
 * @since 1.19.0
 */
export declare function mapWithIndex<K extends string, A, B>(f: (k: K, a: A) => B): (fa: Record<K, A>) => Record<K, B>;
/**
 * @since 1.19.0
 */
export declare function reduceWithIndex<K extends string, A, B>(b: B, f: (k: K, b: B, a: A) => B): (fa: Record<K, A>) => B;
/**
 * @since 1.19.0
 */
export declare function foldMapWithIndex<M>(M: Monoid<M>): <K extends string, A>(f: (k: K, a: A) => M) => (fa: Record<K, A>) => M;
/**
 * @since 1.19.0
 */
export declare function reduceRightWithIndex<K extends string, A, B>(b: B, f: (k: K, a: A, b: B) => B): (fa: Record<K, A>) => B;
/**
 * @since 1.19.0
 */
export declare function hasOwnProperty<K extends string, A>(k: K, d: Record<K, A>): boolean;
/**
 * @since 1.19.0
 */
export declare function traverseWithIndex<F extends URIS3>(F: Applicative3<F>): <K extends string, U, L, A, B>(f: (k: K, a: A) => Kind3<F, U, L, B>) => (ta: Record<K, A>) => Kind3<F, U, L, Record<K, B>>;
export declare function traverseWithIndex<F extends URIS2>(F: Applicative2<F>): <K extends string, L, A, B>(f: (k: K, a: A) => Kind2<F, L, B>) => (ta: Record<K, A>) => Kind2<F, L, Record<K, B>>;
export declare function traverseWithIndex<F extends URIS2, L>(F: Applicative2C<F, L>): <K extends string, A, B>(f: (k: K, a: A) => Kind2<F, L, B>) => (ta: Record<K, A>) => Kind2<F, L, Record<K, B>>;
export declare function traverseWithIndex<F extends URIS>(F: Applicative1<F>): <K extends string, A, B>(f: (k: K, a: A) => Kind<F, B>) => (ta: Record<K, A>) => Kind<F, Record<K, B>>;
export declare function traverseWithIndex<F>(F: Applicative<F>): <K extends string, A, B>(f: (k: K, a: A) => HKT<F, B>) => (ta: Record<K, A>) => HKT<F, Record<K, B>>;
/**
 * @since 1.19.0
 */
export declare function traverse2v<F extends URIS3>(F: Applicative3<F>): <U, L, A, B>(f: (a: A) => Kind3<F, U, L, B>) => <K extends string>(ta: Record<K, A>) => Kind3<F, U, L, Record<K, B>>;
export declare function traverse2v<F extends URIS2>(F: Applicative2<F>): <L, A, B>(f: (a: A) => Kind2<F, L, B>) => <K extends string>(ta: Record<K, A>) => Kind2<F, L, Record<K, B>>;
export declare function traverse2v<F extends URIS2, L>(F: Applicative2C<F, L>): <A, B>(f: (a: A) => Kind2<F, L, B>) => <K extends string>(ta: Record<K, A>) => Kind2<F, L, Record<K, B>>;
export declare function traverse2v<F extends URIS>(F: Applicative1<F>): <A, B>(f: (a: A) => Kind<F, B>) => <K extends string>(ta: Record<K, A>) => Kind<F, Record<K, B>>;
export declare function traverse2v<F>(F: Applicative<F>): <A, B>(f: (a: A) => HKT<F, B>) => <K extends string>(ta: Record<K, A>) => HKT<F, Record<K, B>>;
/**
 * @since 1.19.0
 */
export declare const record: FunctorWithIndex1<URI, string> & Foldable2v1<URI> & TraversableWithIndex1<URI, string> & Compactable1<URI> & FilterableWithIndex1<URI, string> & Witherable1<URI> & FoldableWithIndex1<URI, string>;
declare const reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => (fa: Record<string, A>) => B;
export { reduceRight };
