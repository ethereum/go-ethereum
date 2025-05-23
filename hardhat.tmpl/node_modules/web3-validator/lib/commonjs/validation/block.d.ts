export declare const isBlockNumber: (value: string | number | bigint) => boolean;
/**
 * Returns true if the given blockNumber is 'latest', 'pending', 'earliest, 'safe' or 'finalized'
 */
export declare const isBlockTag: (value: string) => boolean;
/**
 * Returns true if given value is valid hex string and not negative, or is a valid BlockTag
 */
export declare const isBlockNumberOrTag: (value: string | number | bigint) => boolean;
