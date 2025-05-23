import { EthExecutionAPI } from './eth_execution_api.js';
import { AccountObject, Address, BlockNumberOrTag, Eip712TypedData, HexString256Bytes, HexString32Bytes, TransactionInfo, Uint } from '../eth_types.js';
export type Web3EthExecutionAPI = EthExecutionAPI & {
    eth_pendingTransactions: () => TransactionInfo[];
    eth_requestAccounts: () => Address[];
    eth_chainId: () => Uint;
    web3_clientVersion: () => string;
    eth_getProof: (address: Address, storageKeys: HexString32Bytes[], blockNumber: BlockNumberOrTag) => AccountObject;
    eth_signTypedData: (address: Address, typedData: Eip712TypedData, useLegacy: true) => HexString256Bytes;
    eth_signTypedData_v4: (address: Address, typedData: Eip712TypedData, useLegacy: false | undefined) => HexString256Bytes;
};
