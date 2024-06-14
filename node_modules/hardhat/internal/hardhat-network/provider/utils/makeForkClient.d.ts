import { HttpProvider } from "../../../core/providers/http";
import { JsonRpcClient } from "../../jsonrpc/client";
import { ForkConfig } from "../node-types";
export declare function makeForkProvider(forkConfig: ForkConfig): Promise<{
    forkProvider: HttpProvider;
    networkId: number;
    forkBlockNumber: bigint;
    latestBlockNumber: bigint;
    maxReorg: bigint;
}>;
export declare function makeForkClient(forkConfig: ForkConfig, forkCachePath?: string): Promise<{
    forkClient: JsonRpcClient;
    forkBlockNumber: bigint;
    forkBlockTimestamp: number;
    forkBlockHash: string;
    forkBlockStateRoot: string;
}>;
export declare function getLastSafeBlockNumber(latestBlockNumber: bigint, maxReorg: bigint): bigint;
//# sourceMappingURL=makeForkClient.d.ts.map