/// <reference types="node" />
import { ConfigExtender, EnvironmentExtender, HardhatRuntimeEnvironment, ProviderExtender } from "../types";
import { VarsManagerSetup } from "./core/vars/vars-manager-setup";
import { VarsManager } from "./core/vars/vars-manager";
import { TasksDSL } from "./core/tasks/dsl";
export type GlobalWithHardhatContext = typeof global & {
    __hardhatContext: HardhatContext;
};
export declare class HardhatContext {
    constructor();
    static isCreated(): boolean;
    static createHardhatContext(): HardhatContext;
    static getHardhatContext(): HardhatContext;
    static deleteHardhatContext(): void;
    readonly tasksDSL: TasksDSL;
    readonly environmentExtenders: EnvironmentExtender[];
    environment?: HardhatRuntimeEnvironment;
    readonly providerExtenders: ProviderExtender[];
    varsManager: VarsManager | VarsManagerSetup;
    readonly configExtenders: ConfigExtender[];
    private _filesLoadedBeforeConfig?;
    private _filesLoadedAfterConfig?;
    setHardhatRuntimeEnvironment(env: HardhatRuntimeEnvironment): void;
    getHardhatRuntimeEnvironment(): HardhatRuntimeEnvironment;
    setConfigLoadingAsStarted(): void;
    setConfigLoadingAsFinished(): void;
    getFilesLoadedDuringConfig(): string[];
}
//# sourceMappingURL=context.d.ts.map