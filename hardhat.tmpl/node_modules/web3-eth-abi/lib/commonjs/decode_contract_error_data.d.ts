import { Eip838ExecutionError } from 'web3-errors';
import { AbiErrorFragment } from 'web3-types';
export declare const decodeContractErrorData: (errorsAbi: AbiErrorFragment[], error: Eip838ExecutionError) => void;
