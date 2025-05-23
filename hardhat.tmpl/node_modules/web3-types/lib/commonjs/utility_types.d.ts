import { Numbers } from './primitives_types.js';
export type Mutable<T> = {
    -readonly [P in keyof T]: T[P];
};
export type ConnectionEvent = {
    code: number;
    reason: string;
    wasClean?: boolean;
};
export type Optional<T, K extends keyof T> = Pick<Partial<T>, K> & Omit<T, K>;
export type EncodingTypes = Numbers | boolean | Numbers[] | boolean[];
export type TypedObject = {
    type: string;
    value: EncodingTypes;
};
export type TypedObjectAbbreviated = {
    t: string;
    v: EncodingTypes;
};
export type Sha3Input = TypedObject | TypedObjectAbbreviated | Numbers | boolean | object;
export type IndexKeysForArray<A extends readonly unknown[]> = Exclude<keyof A, keyof []>;
export type ArrayToIndexObject<T extends ReadonlyArray<unknown>> = {
    [K in IndexKeysForArray<T>]: T[K];
};
type _Grow<T, A extends Array<T>> = ((x: T, ...xs: A) => void) extends (...a: infer X) => void ? X : never;
export type GrowToSize<T, A extends Array<T>, N extends number> = {
    0: A;
    1: GrowToSize<T, _Grow<T, A>, N>;
}[A['length'] extends N ? 0 : 1];
export type FixedSizeArray<T, N extends number> = GrowToSize<T, [], N>;
export {};
