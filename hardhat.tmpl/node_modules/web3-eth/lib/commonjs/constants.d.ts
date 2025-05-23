import { AbiEventFragment, FMT_BYTES, FMT_NUMBER } from 'web3-types';
export declare const ALL_EVENTS = "ALLEVENTS";
export declare const ALL_EVENTS_ABI: AbiEventFragment & {
    signature: string;
};
export declare const NUMBER_DATA_FORMAT: {
    readonly bytes: FMT_BYTES.HEX;
    readonly number: FMT_NUMBER.NUMBER;
};
