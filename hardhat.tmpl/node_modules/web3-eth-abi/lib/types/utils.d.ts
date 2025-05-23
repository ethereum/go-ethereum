import { AbiInput, AbiCoderStruct, AbiFragment, AbiParameter, AbiStruct, AbiEventFragment, AbiFunctionFragment, AbiConstructorFragment } from 'web3-types';
export declare const isAbiFragment: (item: unknown) => item is AbiFragment;
export declare const isAbiErrorFragment: (item: unknown) => item is AbiEventFragment;
export declare const isAbiEventFragment: (item: unknown) => item is AbiEventFragment;
export declare const isAbiFunctionFragment: (item: unknown) => item is AbiFunctionFragment;
export declare const isAbiConstructorFragment: (item: unknown) => item is AbiConstructorFragment;
/**
 * Check if type is simplified struct format
 */
export declare const isSimplifiedStructFormat: (type: string | Partial<AbiParameter> | Partial<AbiInput>) => type is Omit<AbiParameter, "components" | "name">;
/**
 * Maps the correct tuple type and name when the simplified format in encode/decodeParameter is used
 */
export declare const mapStructNameAndType: (structName: string) => AbiStruct;
/**
 * Maps the simplified format in to the expected format of the ABICoder
 */
export declare const mapStructToCoderFormat: (struct: AbiStruct) => Array<AbiCoderStruct>;
/**
 * Map types if simplified format is used
 */
export declare const mapTypes: (types: AbiInput[]) => Array<string | AbiParameter | Record<string, unknown>>;
/**
 * returns true if input is a hexstring and is odd-lengthed
 */
export declare const isOddHexstring: (param: unknown) => boolean;
/**
 * format odd-length bytes to even-length
 */
export declare const formatOddHexstrings: (param: string) => string;
/**
 * Handle some formatting of params for backwards compatibility with Ethers V4
 */
export declare const formatParam: (type: string, _param: unknown) => unknown;
/**
 *  used to flatten json abi inputs/outputs into an array of type-representing-strings
 */
export declare const flattenTypes: (includeTuple: boolean, puts: ReadonlyArray<AbiParameter>) => string[];
/**
 * Should be used to create full function/event name from json abi
 * returns a string
 */
export declare const jsonInterfaceMethodToString: (json: AbiFragment) => string;
//# sourceMappingURL=utils.d.ts.map