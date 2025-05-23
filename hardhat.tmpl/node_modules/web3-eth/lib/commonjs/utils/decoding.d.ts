import { AbiEventFragment, LogsInput, DataFormat, EventLog, ContractAbiWithSignature } from 'web3-types';
export declare const decodeEventABI: (event: AbiEventFragment & {
    signature: string;
}, data: LogsInput, jsonInterface: ContractAbiWithSignature, returnFormat?: DataFormat) => EventLog;
