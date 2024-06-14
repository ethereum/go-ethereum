import { Config, Services } from '../../typechain/types';
export type OutputTransformer = (output: string, services: Services, cfg: Config) => string;
export declare const outputTransformers: OutputTransformer[];
