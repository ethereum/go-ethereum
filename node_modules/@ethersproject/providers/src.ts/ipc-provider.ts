"use strict";

import { connect } from "net";

import { defineReadOnly } from "@ethersproject/properties";
import { Networkish } from "@ethersproject/networks";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { JsonRpcProvider } from "./json-rpc-provider";


export class IpcProvider extends JsonRpcProvider {
    readonly path: string;

    constructor(path: string, network?: Networkish) {
        if (path == null) {
            logger.throwError("missing path", Logger.errors.MISSING_ARGUMENT, { arg: "path" });
        }

        super("ipc://" + path, network);

        defineReadOnly(this, "path", path);
    }

    // @TODO: Create a connection to the IPC path and use filters instead of polling for block

    send(method: string, params: Array<any>): Promise<any> {
        // This method is very simple right now. We create a new socket
        // connection each time, which may be slower, but the main
        // advantage we are aiming for now is security. This simplifies
        // multiplexing requests (since we do not need to multiplex).

        let payload = JSON.stringify({
            method: method,
            params: params,
            id: 42,
            jsonrpc: "2.0"
        });

        return new Promise((resolve, reject) => {
            let response = Buffer.alloc(0);

            let stream = connect(this.path);

            stream.on("data", (data) => {
                response = Buffer.concat([ response, data ]);
            });

            stream.on("end", () => {
                try {
                    resolve(JSON.parse(response.toString()).result);
                    // @TODO: Better pull apart the error
                    stream.destroy();
                } catch (error) {
                    reject(error);
                    stream.destroy();
                }
            });

            stream.on("error", (error) => {
                reject(error);
                stream.destroy();
            });

            stream.write(payload);
            stream.end();
        });
    }
}
