import { CodegenConfig, PublicConfig } from './types';
interface Result {
    filesGenerated: number;
}
export declare const DEFAULT_FLAGS: CodegenConfig;
export declare function runTypeChain(publicConfig: PublicConfig): Promise<Result>;
export {};
