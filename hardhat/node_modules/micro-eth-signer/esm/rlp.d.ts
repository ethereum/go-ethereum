import * as P from 'micro-packed';
export type RLPInput = string | number | Uint8Array | bigint | RLPInput[] | null;
export type InternalRLP = {
    TAG: 'byte';
    data: number;
} | {
    TAG: 'complex';
    data: {
        TAG: 'string';
        data: Uint8Array;
    } | {
        TAG: 'list';
        data: InternalRLP[];
    };
};
/**
 * RLP parser.
 * Real type of rlp is `Item = Uint8Array | Item[]`.
 * Strings/number encoded to Uint8Array, but not decoded back: type information is lost.
 */
export declare const RLP: P.CoderType<RLPInput>;
//# sourceMappingURL=rlp.d.ts.map