import type { Addressable } from "ethers";
export interface BalanceChangeOptions {
    includeFee?: boolean;
}
export declare function getAddresses(accounts: Array<Addressable | string>): Promise<string[]>;
export declare function getBalances(accounts: Array<Addressable | string>, blockNumber?: number): Promise<bigint[]>;
//# sourceMappingURL=balance.d.ts.map