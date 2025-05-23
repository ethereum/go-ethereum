import { Web3Context } from 'web3-core';
import { EthExecutionAPI, Numbers, DataFormat, FormatType } from 'web3-types';
import { InternalTransaction } from '../types.js';
export declare function getTransactionGasPricing<ReturnFormat extends DataFormat>(transaction: InternalTransaction, web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<FormatType<{
    gasPrice?: Numbers;
    maxPriorityFeePerGas?: Numbers;
    maxFeePerGas?: Numbers;
}, ReturnFormat> | undefined>;
