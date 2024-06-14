import { IgnitionModule, IgnitionModuleResult } from "./types/module";
import { IgnitionModuleBuilder } from "./types/module-builder";
/**
 * Construct a module definition that can be deployed through Ignition.
 *
 * @param moduleId - the id of the module
 * @param moduleDefintionFunction - a function accepting the
 * IgnitionModuleBuilder to configure the deployment
 * @returns a module definition
 *
 * @beta
 */
export declare function buildModule<ModuleIdT extends string, ContractNameT extends string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT>>(moduleId: ModuleIdT, moduleDefintionFunction: (m: IgnitionModuleBuilder) => IgnitionModuleResultsT): IgnitionModule<ModuleIdT, ContractNameT, IgnitionModuleResultsT>;
//# sourceMappingURL=build-module.d.ts.map