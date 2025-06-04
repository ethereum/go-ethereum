import type { ContractABI, HintFn, HookFn } from './decoder.ts';
export declare function addHint<T extends ContractABI>(abi: ContractABI, name: string, fn: HintFn): T;
export declare function addHints<T extends ContractABI>(abi: T, map: Record<string, HintFn>): T;
export declare function addHook<T extends ContractABI>(abi: T, name: string, fn: HookFn): T;
//# sourceMappingURL=common.d.ts.map