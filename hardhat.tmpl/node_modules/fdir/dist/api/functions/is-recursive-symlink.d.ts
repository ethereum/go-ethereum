import { WalkerState } from "../../types";
type IsRecursiveSymlinkFunction = (state: WalkerState, path: string, resolved: string, callback: (result: boolean) => void) => void;
export declare const isRecursiveAsync: IsRecursiveSymlinkFunction;
export declare function isRecursive(state: WalkerState, path: string, resolved: string): boolean;
export {};
