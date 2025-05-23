/// <reference types="node" />
import { WalkerState } from "../../types";
import fs from "fs";
export type WalkDirectoryFunction = (state: WalkerState, crawlPath: string, directoryPath: string, depth: number, callback: (entries: fs.Dirent[], directoryPath: string, depth: number) => void) => void;
export declare function build(isSynchronous: boolean): WalkDirectoryFunction;
