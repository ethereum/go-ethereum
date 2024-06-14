/// <reference types="node" />
import * as t from "io-ts";
export declare const rpcQuantity: t.Type<bigint, bigint, unknown>;
export declare const rpcData: t.Type<Buffer, Buffer, unknown>;
export declare const rpcHash: t.Type<Buffer, Buffer, unknown>;
export declare const rpcStorageSlot: t.Type<bigint, bigint, unknown>;
export declare const rpcStorageSlotHexString: t.Type<string, string, unknown>;
export declare const rpcAddress: t.Type<Buffer, Buffer, unknown>;
export declare const rpcUnsignedInteger: t.Type<number, number, unknown>;
export declare const rpcQuantityAsNumber: t.Type<bigint, bigint, unknown>;
export declare const rpcFloat: t.Type<number, number, unknown>;
/**
 * Transforms a QUANTITY into a number. It should only be used if you are 100% sure that the value
 * fits in a number.
 */
export declare function rpcQuantityToNumber(quantity: string): number;
export declare function rpcQuantityToBigInt(quantity: string): bigint;
export declare function numberToRpcQuantity(n: number | bigint): string;
export declare function numberToRpcStorageSlot(n: number | bigint): string;
/**
 * Transforms a DATA into a number. It should only be used if you are 100% sure that the data
 * represents a value fits in a number.
 */
export declare function rpcDataToNumber(data: string): number;
export declare function rpcDataToBigInt(data: string): bigint;
export declare function bufferToRpcData(buffer: Uint8Array, padToBytes?: number): string;
export declare function rpcDataToBuffer(data: string): Buffer;
//# sourceMappingURL=base-types.d.ts.map