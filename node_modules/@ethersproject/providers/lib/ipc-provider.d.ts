import { Networkish } from "@ethersproject/networks";
import { JsonRpcProvider } from "./json-rpc-provider";
export declare class IpcProvider extends JsonRpcProvider {
    readonly path: string;
    constructor(path: string, network?: Networkish);
    send(method: string, params: Array<any>): Promise<any>;
}
//# sourceMappingURL=ipc-provider.d.ts.map