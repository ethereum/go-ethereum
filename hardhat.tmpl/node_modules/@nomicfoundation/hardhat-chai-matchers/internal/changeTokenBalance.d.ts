/// <reference types="chai" />
import type EthersT from "ethers";
import type { Addressable, BaseContract, BaseContractMethod, BigNumberish, ContractTransactionResponse } from "ethers";
type TransactionResponse = EthersT.TransactionResponse;
export type Token = BaseContract & {
    balanceOf: BaseContractMethod<[string], bigint, bigint>;
    name: BaseContractMethod<[], string, string>;
    transfer: BaseContractMethod<[
        string,
        BigNumberish
    ], boolean, ContractTransactionResponse>;
    symbol: BaseContractMethod<[], string, string>;
};
export declare function supportChangeTokenBalance(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
export declare function getBalanceChange(transaction: TransactionResponse | Promise<TransactionResponse>, token: Token, account: Addressable | string): Promise<bigint>;
export declare function clearTokenDescriptionsCache(): void;
export {};
//# sourceMappingURL=changeTokenBalance.d.ts.map