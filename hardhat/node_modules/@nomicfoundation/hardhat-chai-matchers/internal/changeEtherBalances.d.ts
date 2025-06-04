/// <reference types="chai" />
import type { Addressable, TransactionResponse } from "ethers";
import type { BalanceChangeOptions } from "./misc/balance";
export declare function supportChangeEtherBalances(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
export declare function getBalanceChanges(transaction: TransactionResponse | Promise<TransactionResponse>, accounts: Array<Addressable | string>, options?: BalanceChangeOptions): Promise<bigint[]>;
//# sourceMappingURL=changeEtherBalances.d.ts.map