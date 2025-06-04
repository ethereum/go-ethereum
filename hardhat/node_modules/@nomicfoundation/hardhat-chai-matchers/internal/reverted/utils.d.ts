import type EthersT from "ethers";
/**
 * Try to obtain the return data of a transaction from the given value.
 *
 * If the value is an error but it doesn't have data, we assume it's not related
 * to a reverted transaction and we re-throw it.
 */
export declare function getReturnDataFromError(error: any): string;
type DecodedReturnData = {
    kind: "Error";
    reason: string;
} | {
    kind: "Empty";
} | {
    kind: "Panic";
    code: bigint;
    description: string;
} | {
    kind: "Custom";
    id: string;
    data: string;
};
export declare function decodeReturnData(returnData: string): DecodedReturnData;
/**
 * Takes an ethers result object and converts it into a (potentially nested) array.
 *
 * For example, given this error:
 *
 *   struct Point(uint x, uint y)
 *   error MyError(string, Point)
 *
 *   revert MyError("foo", Point(1, 2))
 *
 * The resulting array will be: ["foo", [1n, 2n]]
 */
export declare function resultToArray(result: EthersT.Result): any[];
export declare function parseBytes32String(v: string): string;
export {};
//# sourceMappingURL=utils.d.ts.map