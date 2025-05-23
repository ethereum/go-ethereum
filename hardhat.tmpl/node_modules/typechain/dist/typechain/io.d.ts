import { Config, FileDescription, Output, Services } from './types';
export declare function processOutput(services: Services, cfg: Config, output: Output): number;
export declare function loadFileDescriptions(services: Services, files: string[]): FileDescription[];
export declare function skipEmptyAbis(paths: string[]): string[];
