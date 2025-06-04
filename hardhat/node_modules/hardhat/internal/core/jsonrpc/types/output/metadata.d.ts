export interface HardhatMetadata {
    clientVersion: string;
    chainId: number;
    instanceId: string;
    latestBlockNumber: number;
    latestBlockHash: string;
    forkedNetwork?: {
        chainId: number;
        forkBlockNumber: number;
        forkBlockHash: string;
    };
}
//# sourceMappingURL=metadata.d.ts.map