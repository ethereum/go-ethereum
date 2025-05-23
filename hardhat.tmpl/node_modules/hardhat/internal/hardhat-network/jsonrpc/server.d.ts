import { EIP1193Provider, JsonRpcServer as IJsonRpcServer } from "../../../types";
export interface JsonRpcServerConfig {
    hostname: string;
    port: number;
    provider: EIP1193Provider;
}
export declare class JsonRpcServer implements IJsonRpcServer {
    private _config;
    private _httpServer;
    private _wsServer;
    constructor(config: JsonRpcServerConfig);
    getProvider: (name?: string) => EIP1193Provider;
    listen: () => Promise<{
        address: string;
        port: number;
    }>;
    waitUntilClosed: () => Promise<void>;
    close: () => Promise<void>;
}
//# sourceMappingURL=server.d.ts.map