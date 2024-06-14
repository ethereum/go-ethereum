import type { EIP1193Provider } from "hardhat/types";
import type { NumberLike } from "./types";
export declare function getHardhatProvider(): Promise<EIP1193Provider>;
export declare function toNumber(x: NumberLike): number;
export declare function toBigInt(x: NumberLike): bigint;
export declare function toRpcQuantity(x: NumberLike): string;
export declare function assertValidAddress(address: string): void;
export declare function assertHexString(hexString: string): void;
export declare function assertTxHash(hexString: string): void;
export declare function assertNonNegativeNumber(n: bigint): void;
export declare function assertLargerThan(a: bigint, b: bigint, type: string): void;
export declare function assertLargerThan(a: number, b: number, type: string): void;
export declare function toPaddedRpcQuantity(x: NumberLike, bytesLength: number): string;
//# sourceMappingURL=utils.d.ts.map