/// <reference types="node" />
import fs from "fs";
import { WalkerState, Options } from "../../types";
export type ResolveSymlinkFunction = (path: string, state: WalkerState, callback: (stat: fs.Stats, path: string) => void) => void;
export declare function build(options: Options, isSynchronous: boolean): ResolveSymlinkFunction | null;
