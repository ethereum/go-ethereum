import { TransactionForAccessList, AbiFunctionFragment, TransactionWithSenderAPI, TransactionCall, HexString, Address, NonPayableCallOptions, PayableCallOptions, ContractOptions, Numbers, AbiConstructorFragment } from 'web3-types';
import { Web3ContractContext } from './types.js';
export declare const getSendTxParams: ({ abi, params, options, contractOptions, }: {
    abi: AbiFunctionFragment | AbiConstructorFragment;
    params: unknown[];
    options?: (PayableCallOptions | NonPayableCallOptions) & {
        input?: HexString;
        data?: HexString;
        to?: Address;
        dataInputFill?: "input" | "data" | "both";
    };
    contractOptions: ContractOptions;
}) => TransactionCall;
export declare const getEthTxCallParams: ({ abi, params, options, contractOptions, }: {
    abi: AbiFunctionFragment;
    params: unknown[];
    options?: (PayableCallOptions | NonPayableCallOptions) & {
        to?: Address;
        dataInputFill?: "input" | "data" | "both";
    };
    contractOptions: ContractOptions;
}) => TransactionCall;
export declare const getEstimateGasParams: ({ abi, params, options, contractOptions, }: {
    abi: AbiFunctionFragment;
    params: unknown[];
    options?: (PayableCallOptions | NonPayableCallOptions) & {
        dataInputFill?: "input" | "data" | "both";
    };
    contractOptions: ContractOptions;
}) => Partial<TransactionWithSenderAPI>;
export declare const isWeb3ContractContext: (options: unknown) => options is Web3ContractContext;
export declare const getCreateAccessListParams: ({ abi, params, options, contractOptions, }: {
    abi: AbiFunctionFragment;
    params: unknown[];
    options?: (PayableCallOptions | NonPayableCallOptions) & {
        to?: Address;
        dataInputFill?: "input" | "data" | "both";
    };
    contractOptions: ContractOptions;
}) => TransactionForAccessList;
export declare const createContractAddress: (from: Address, nonce: Numbers) => Address;
export declare const create2ContractAddress: (from: Address, salt: HexString, initCode: HexString) => Address;
