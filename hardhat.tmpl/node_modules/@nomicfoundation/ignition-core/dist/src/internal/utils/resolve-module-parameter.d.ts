import { DeploymentParameters } from "../../types/deploy";
import { ModuleParameterRuntimeValue, ModuleParameterType, SolidityParameterType } from "../../types/module";
export declare function resolveModuleParameter(moduleParamRuntimeValue: ModuleParameterRuntimeValue<ModuleParameterType>, context: {
    deploymentParameters: DeploymentParameters;
    accounts: string[];
}): SolidityParameterType;
//# sourceMappingURL=resolve-module-parameter.d.ts.map