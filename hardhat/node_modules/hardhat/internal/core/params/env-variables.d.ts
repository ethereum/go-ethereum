/// <reference types="node" />
import ProcessEnv = NodeJS.ProcessEnv;
import { HardhatArguments, HardhatParamDefinitions } from "../../../types";
export declare function paramNameToEnvVariable(paramName: string): string;
export declare function getEnvVariablesMap(hardhatArguments: HardhatArguments): {
    [envVar: string]: string;
};
export declare function getEnvHardhatArguments(paramDefinitions: HardhatParamDefinitions, envVariables: ProcessEnv): HardhatArguments;
//# sourceMappingURL=env-variables.d.ts.map