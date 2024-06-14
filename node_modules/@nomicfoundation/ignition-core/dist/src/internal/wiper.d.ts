import { DeploymentLoader } from "./deployment-loader/types";
export declare class Wiper {
    private _deploymentLoader;
    constructor(_deploymentLoader: DeploymentLoader);
    wipe(futureId: string): Promise<import("./execution/types/deployment-state").DeploymentState>;
}
//# sourceMappingURL=wiper.d.ts.map