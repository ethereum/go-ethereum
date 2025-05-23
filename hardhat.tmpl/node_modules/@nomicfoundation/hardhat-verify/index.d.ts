import "./internal/type-extensions";
import "./internal/tasks/etherscan";
import "./internal/tasks/sourcify";
import "./internal/tasks/blockscout";
export interface VerifyTaskArgs {
    address?: string;
    constructorArgsParams: string[];
    constructorArgs?: string;
    libraries?: string;
    contract?: string;
    force: boolean;
    listNetworks: boolean;
}
export interface VerificationResponse {
    success: boolean;
    message: string;
}
export interface VerificationSubtask {
    label: string;
    subtaskName: string;
}
//# sourceMappingURL=index.d.ts.map