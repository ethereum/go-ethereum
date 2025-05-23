import { EthExecutionAPI, HexString, Transaction } from 'web3-types';
import { Web3Context } from 'web3-core';
export declare const prepareTransactionForSigning: (transaction: Transaction, web3Context: Web3Context<EthExecutionAPI>, privateKey?: HexString | Uint8Array, fillGasPrice?: boolean, fillGasLimit?: boolean) => Promise<import("web3-eth-accounts").TypedTransaction>;
//# sourceMappingURL=prepare_transaction_for_signing.d.ts.map