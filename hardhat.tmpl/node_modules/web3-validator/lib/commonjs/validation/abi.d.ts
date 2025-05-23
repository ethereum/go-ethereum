import { AbiParameter } from 'web3-types';
import { ShortValidationSchema } from '../types';
export declare const isAbiParameterSchema: (schema: string | ShortValidationSchema | AbiParameter) => schema is AbiParameter;
