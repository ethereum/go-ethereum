import type { SourcifyVerifyResponse } from "./sourcify.types";
import { ContractStatus } from "./sourcify.types";
import { ValidationResponse } from "./utilities";
export declare class Sourcify {
    chainId: number;
    apiUrl: string;
    browserUrl: string;
    constructor(chainId: number, apiUrl: string, browserUrl: string);
    isVerified(address: string): Promise<false | ContractStatus.PERFECT | ContractStatus.PARTIAL>;
    verify(address: string, files: Record<string, string>, chosenContract?: number): Promise<SourcifyResponse>;
    getContractUrl(address: string, contractStatus: ContractStatus.PERFECT | ContractStatus.PARTIAL): string;
}
declare class SourcifyResponse implements ValidationResponse {
    readonly error: string | undefined;
    readonly status: ContractStatus.PERFECT | ContractStatus.PARTIAL | undefined;
    constructor(response: SourcifyVerifyResponse);
    isPending(): boolean;
    isFailure(): boolean;
    isSuccess(): boolean;
    isOk(): this is {
        status: ContractStatus.PERFECT | ContractStatus.PARTIAL;
    };
}
export {};
//# sourceMappingURL=sourcify.d.ts.map