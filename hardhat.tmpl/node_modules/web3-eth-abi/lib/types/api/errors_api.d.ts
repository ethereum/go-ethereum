import { AbiErrorFragment } from 'web3-types';
/**
 * Encodes the error name to its ABI signature, which are the sha3 hash of the error name including input types.
 */
export declare const encodeErrorSignature: (functionName: string | AbiErrorFragment) => string;
//# sourceMappingURL=errors_api.d.ts.map