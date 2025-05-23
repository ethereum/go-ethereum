import { IgnitionError } from "../../errors";
import { DeploymentParameters } from "../../types/deploy";
import { AccountRuntimeValue, ArgumentType, ModuleParameterRuntimeValue, RuntimeValue, SolidityParameterType } from "../../types/module";
/**
 * Given the deployment parameters and a ModuleParameterRuntimeValue,
 * resolve the value for the ModuleParameterRuntimeValue.
 *
 * The logic runs, use the specific module parameter if available,
 * fall back to a globally defined parameter, then finally use
 * the default value. It is possible that the ModuleParameterRuntimeValue
 * has no default value, in which case this function will return undefined.
 */
export declare function resolvePotentialModuleParameterValueFrom(deploymentParameters: DeploymentParameters, moduleRuntimeValue: ModuleParameterRuntimeValue<any>): SolidityParameterType | undefined;
export declare function validateAccountRuntimeValue(arv: AccountRuntimeValue, accounts: string[]): IgnitionError[];
export declare function filterToAccountRuntimeValues(runtimeValues: RuntimeValue[]): AccountRuntimeValue[];
export declare function retrieveNestedRuntimeValues(args: ArgumentType[]): RuntimeValue[];
//# sourceMappingURL=utils.d.ts.map