import type { BigNumber as EthersBigNumberType } from "ethers-v5";
import type { BigNumber as BigNumberJsType } from "bignumber.js";
import type { default as BNType } from "bn.js";
export declare function normalizeToBigInt(source: number | bigint | BNType | EthersBigNumberType | BigNumberJsType | string): bigint;
export declare function isBigNumber(source: any): boolean;
export declare function formatNumberType(n: string | bigint | BNType | EthersBigNumberType | BigNumberJsType): string;
//# sourceMappingURL=bigInt.d.ts.map