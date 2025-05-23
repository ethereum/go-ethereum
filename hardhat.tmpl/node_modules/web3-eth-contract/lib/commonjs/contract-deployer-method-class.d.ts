import { SendTransactionEvents } from 'web3-eth';
import { AbiConstructorFragment, ContractAbi, ContractConstructorArgs, HexString, PayableCallOptions, DataFormat, DEFAULT_RETURN_FORMAT, ContractOptions, TransactionCall } from 'web3-types';
import { Web3PromiEvent } from 'web3-core';
import { NonPayableTxOptions, PayableTxOptions } from './types.js';
import { Contract } from './contract.js';
export type ContractDeploySend<Abi extends ContractAbi> = Web3PromiEvent<Contract<Abi>, SendTransactionEvents<DataFormat>>;
export declare class DeployerMethodClass<FullContractAbi extends ContractAbi> {
    parent: Contract<FullContractAbi>;
    deployOptions: {
        /**
         * The byte code of the contract.
         */
        data?: HexString;
        input?: HexString;
        /**
         * The arguments which get passed to the constructor on deployment.
         */
        arguments?: ContractConstructorArgs<FullContractAbi>;
    } | undefined;
    protected readonly args: never[] | ContractConstructorArgs<FullContractAbi>;
    protected readonly constructorAbi: AbiConstructorFragment;
    protected readonly contractOptions: ContractOptions;
    protected readonly deployData?: string;
    protected _contractMethodDeploySend(tx: TransactionCall): Web3PromiEvent<Contract<FullContractAbi>, SendTransactionEvents<DataFormat>>;
    constructor(parent: Contract<FullContractAbi>, deployOptions: {
        /**
         * The byte code of the contract.
         */
        data?: HexString;
        input?: HexString;
        /**
         * The arguments which get passed to the constructor on deployment.
         */
        arguments?: ContractConstructorArgs<FullContractAbi>;
    } | undefined);
    send(options?: PayableTxOptions): ContractDeploySend<FullContractAbi>;
    populateTransaction(txOptions?: PayableTxOptions | NonPayableTxOptions): TransactionCall;
    protected calculateDeployParams(): {
        args: never[] | NonNullable<ContractConstructorArgs<FullContractAbi>>;
        abi: AbiConstructorFragment;
        contractOptions: ContractOptions;
        deployData: string | undefined;
    };
    estimateGas<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(options?: PayableCallOptions, returnFormat?: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
    encodeABI(): string;
    decodeData(data: HexString): {
        __method__: string;
        __length__: number;
    };
}
