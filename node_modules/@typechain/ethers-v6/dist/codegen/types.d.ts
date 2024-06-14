import { AbiOutputParameter, AbiParameter, EvmOutputType, EvmType, TupleType } from 'typechain';
interface GenerateTypeOptions {
    returnResultObject?: boolean;
    useStructs?: boolean;
    includeLabelsInTupleTypes?: boolean;
}
export declare function generateInputTypes(input: Array<AbiParameter>, options: GenerateTypeOptions): string;
export declare function generateOutputTypes(options: GenerateTypeOptions, outputs: Array<AbiOutputParameter>): string;
export declare function generateInputType(options: GenerateTypeOptions, evmType: EvmType): string;
export declare function generateOutputType(options: GenerateTypeOptions, evmType: EvmOutputType): string;
export declare function generateObjectTypeLiteral(tuple: TupleType, generator: (evmType: EvmType) => string): string;
export declare function generateInputComplexTypeAsTuple(components: AbiParameter[], options: GenerateTypeOptions): string;
/**
 * Always return an array type; if there are named outputs, merge them to that type
 * this generates slightly better typings fixing: https://github.com/ethereum-ts/TypeChain/issues/232
 **/
export declare function generateOutputComplexType(components: AbiOutputParameter[], options: GenerateTypeOptions): string;
export declare function generateOutputComplexTypeAsTuple(components: AbiOutputParameter[], options: GenerateTypeOptions): string;
export declare function generateOutputComplexTypesAsObject(components: AbiOutputParameter[], options: GenerateTypeOptions): string | undefined;
export {};
