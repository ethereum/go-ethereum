/// <reference types="chai" />
import type { Addressable, TransactionResponse } from "ethers";
import type { BalanceChangeOptions } from "./misc/balance";
export declare function supportChangeEtherBalance(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
export declare function getBalanceChange(transaction: TransactionResponse | Promise<TransactionResponse> | (() => Promise<TransactionResponse> | TransactionResponse), account: Addressable | string, options?: BalanceChangeOptions): Promise<bigint>;
//# sourceMappingURL=changeEtherBalance.d.ts.map