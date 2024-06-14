import { Eq } from './Eq';
import { Filterable2 } from './Filterable';
import { FilterableWithIndex2C } from './FilterableWithIndex';
import { Foldable2v, Foldable2v1, Foldable2v2, Foldable2v3 } from './Foldable2v';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monoid } from './Monoid';
import { Option } from './Option';
import { Ord } from './Ord';
import { Semigroup } from './Semigroup';
import { Show } from './Show';
import { TraversableWithIndex2C } from './TraversableWithIndex';
import { Unfoldable, Unfoldable1 } from './Unfoldable';
import { Witherable2C } from './Witherable';
declare module './HKT' {
    interface URItoKind2<L, A> {
        Map: Map<L, A>;
    }
}
export declare const URI = "Map";
export declare type URI = typeof URI;
/**
 * @since 1.17.0
 */
export declare const getShow: <K, A>(SK: Show<K>, SA: Show<A>) => Show<Map<K, A>>;
/**
 * Calculate the number of key/value pairs in a map
 *
 * @since 1.14.0
 */
export declare const size: <K, A>(d: Map<K, A>) => number;
/**
 * Test whether or not a map is empty
 *
 * @since 1.14.0
 */
export declare const isEmpty: <K, A>(d: Map<K, A>) => boolean;
/**
 * Test whether or not a key exists in a map
 *
 * @since 1.14.0
 */
export declare const member: <K>(E: Eq<K>) => <A>(k: K, m: Map<K, A>) => boolean;
/**
 * Test whether or not a value is a member of a map
 *
 * @since 1.14.0
 */
export declare const elem: <A>(E: Eq<A>) => <K>(a: A, m: Map<K, A>) => boolean;
/**
 * Get a sorted array of the keys contained in a map
 *
 * @since 1.14.0
 */
export declare const keys: <K>(O: Ord<K>) => <A>(m: Map<K, A>) => K[];
/**
 * Get a sorted array of the values contained in a map
 *
 * @since 1.14.0
 */
export declare const values: <A>(O: Ord<A>) => <K>(m: Map<K, A>) => A[];
/**
 * @since 1.14.0
 */
export declare const collect: <K>(O: Ord<K>) => <A, B>(m: Map<K, A>, f: (k: K, a: A) => B) => B[];
/**
 * Get a sorted of the key/value pairs contained in a map
 *
 * @since 1.14.0
 */
export declare const toArray: <K>(O: Ord<K>) => <A>(m: Map<K, A>) => [K, A][];
/**
 * Unfolds a map into a list of key/value pairs
 *
 * @since 1.14.0
 */
export declare function toUnfoldable<K, F extends URIS>(O: Ord<K>, unfoldable: Unfoldable1<F>): <A>(d: Map<K, A>) => Kind<F, [K, A]>;
export declare function toUnfoldable<K, F>(O: Ord<K>, unfoldable: Unfoldable<F>): <A>(d: Map<K, A>) => HKT<F, [K, A]>;
/**
 * Use `insertAt`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare const insert: <K>(E: Eq<K>) => <A>(k: K, a: A, m: Map<K, A>) => Map<K, A>;
/**
 * Use `deleteAt`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare const remove: <K>(E: Eq<K>) => <A>(k: K, m: Map<K, A>) => Map<K, A>;
/**
 * Delete a key and value from a map, returning the value as well as the subsequent map
 *
 * @since 1.14.0
 */
export declare const pop: <K>(E: Eq<K>) => <A>(k: K, m: Map<K, A>) => Option<[A, Map<K, A>]>;
/**
 * Lookup the value for a key in a `Map`.
 * If the result is a `Some`, the existing key is also returned.
 *
 * @since 1.14.0
 */
export declare const lookupWithKey: <K>(E: Eq<K>) => <A>(k: K, m: Map<K, A>) => Option<[K, A]>;
/**
 * Lookup the value for a key in a `Map`.
 *
 * @since 1.14.0
 */
export declare const lookup: <K>(E: Eq<K>) => <A>(k: K, m: Map<K, A>) => Option<A>;
/**
 * Test whether or not one Map contains all of the keys and values contained in another Map
 *
 * @since 1.14.0
 */
export declare const isSubmap: <K, A>(EK: Eq<K>, EA: Eq<A>) => (d1: Map<K, A>, d2: Map<K, A>) => boolean;
/**
 * @since 1.14.0
 */
export declare const empty: Map<never, never>;
/**
 * Use `getEq`
 *
 * @since 1.14.0
 * @deprecated
 */
export declare const getSetoid: <K, A>(EK: Eq<K>, EA: Eq<A>) => Eq<Map<K, A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<K, A>(EK: Eq<K>, EA: Eq<A>): Eq<Map<K, A>>;
/**
 * Gets `Monoid` instance for Maps given `Semigroup` instance for their values
 *
 * @since 1.14.0
 */
export declare const getMonoid: <K, A>(EK: Eq<K>, EA: Semigroup<A>) => Monoid<Map<K, A>>;
/**
 * Create a map with one key/value pair
 *
 * @since 1.14.0
 */
export declare const singleton: <K, A>(k: K, a: A) => Map<K, A>;
/**
 * Create a map from a foldable collection of key/value pairs, using the
 * specified function to combine values for duplicate keys.
 *
 * @since 1.14.0
 */
export declare function fromFoldable<K, F extends URIS3>(E: Eq<K>, F: Foldable2v3<F>): <U, L, A>(ta: Kind3<F, U, L, [K, A]>, onConflict: (existing: A, a: A) => A) => Map<K, A>;
export declare function fromFoldable<K, F extends URIS2>(E: Eq<K>, F: Foldable2v2<F>): <L, A>(ta: Kind2<F, L, [K, A]>, onConflict: (existing: A, a: A) => A) => Map<K, A>;
export declare function fromFoldable<K, F extends URIS>(E: Eq<K>, F: Foldable2v1<F>): <A>(ta: Kind<F, [K, A]>, onConflict: (existing: A, a: A) => A) => Map<K, A>;
export declare function fromFoldable<K, F>(E: Eq<K>, F: Foldable2v<F>): <A>(ta: HKT<F, [K, A]>, onConflict: (existing: A, a: A) => A) => Map<K, A>;
/**
 * @since 1.14.0
 */
export declare const getFilterableWithIndex: <K>() => FilterableWithIndex2C<"Map", K, K>;
/**
 * @since 1.14.0
 */
export declare const getWitherable: <K>(O: Ord<K>) => Witherable2C<"Map", K>;
/**
 * @since 1.14.0
 */
export declare const getTraversableWithIndex: <K>(O: Ord<K>) => TraversableWithIndex2C<"Map", K, K>;
/**
 * @since 1.14.0
 */
export declare const map: Filterable2<URI>;
/**
 * Insert or replace a key/value pair in a map
 *
 * @since 1.19.0
 */
export declare function insertAt<K>(E: Eq<K>): <A>(k: K, a: A) => (m: Map<K, A>) => Map<K, A>;
/**
 * Delete a key and value from a map
 *
 * @since 1.19.0
 */
export declare function deleteAt<K>(E: Eq<K>): (k: K) => <A>(m: Map<K, A>) => Map<K, A>;
/**
 * @since 1.19.0
 */
export declare function updateAt<K>(E: Eq<K>): <A>(k: K, a: A) => (m: Map<K, A>) => Option<Map<K, A>>;
/**
 * @since 1.19.0
 */
export declare function modifyAt<K>(E: Eq<K>): <A>(k: K, f: (a: A) => A) => (m: Map<K, A>) => Option<Map<K, A>>;
