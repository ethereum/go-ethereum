import type { Client as ClientT } from "undici";
import type WsT from "ws";

import debug from "debug";
import http, { Server } from "http";
import { AddressInfo } from "net";

import {
  EIP1193Provider,
  JsonRpcServer as IJsonRpcServer,
} from "../../../types";
import { HttpProvider } from "../../core/providers/http";

import { JsonRpcHandler } from "./handler";

const log = debug("hardhat:core:hardhat-network:jsonrpc");

export interface JsonRpcServerConfig {
  hostname: string;
  port: number;

  provider: EIP1193Provider;
}

export class JsonRpcServer implements IJsonRpcServer {
  private _config: JsonRpcServerConfig;
  private _httpServer: Server;
  private _wsServer: WsT.Server;

  constructor(config: JsonRpcServerConfig) {
    const { Server: WSServer } = require("ws") as typeof WsT;

    this._config = config;

    const handler = new JsonRpcHandler(config.provider);

    this._httpServer = http.createServer();
    this._wsServer = new WSServer({
      server: this._httpServer,
    });

    this._httpServer.on("request", handler.handleHttp);
    this._wsServer.on("connection", handler.handleWs);
  }

  public getProvider = (name = "json-rpc"): EIP1193Provider => {
    const { Client } = require("undici") as { Client: typeof ClientT };
    const { address, port } = this._httpServer.address() as AddressInfo;

    const dispatcher = new Client(`http://${address}:${port}/`, {
      keepAliveTimeout: 10,
      keepAliveMaxTimeout: 10,
    });

    return new HttpProvider(
      `http://${address}:${port}/`,
      name,
      {},
      20000,
      dispatcher
    );
  };

  public listen = (): Promise<{ address: string; port: number }> => {
    return new Promise((resolve) => {
      log(`Starting JSON-RPC server on port ${this._config.port}`);
      this._httpServer.listen(this._config.port, this._config.hostname, () => {
        // We get the address and port directly from the server in order to handle random port allocation with `0`.
        const address = this._httpServer.address() as AddressInfo; // TCP sockets return AddressInfo
        resolve(address);
      });
    });
  };

  public waitUntilClosed = async () => {
    const httpServerClosed = new Promise((resolve) => {
      this._httpServer.once("close", resolve);
    });

    const wsServerClosed = new Promise((resolve) => {
      this._wsServer.once("close", resolve);
    });

    await Promise.all([httpServerClosed, wsServerClosed]);
  };

  public close = async () => {
    await Promise.all([
      new Promise<void>((resolve, reject) => {
        log("Closing JSON-RPC server");
        this._httpServer.close((err) => {
          if (err !== null && err !== undefined) {
            log("Failed to close JSON-RPC server");
            reject(err);
            return;
          }

          log("JSON-RPC server closed");
          resolve();
        });
      }),
      new Promise<void>((resolve, reject) => {
        log("Closing websocket server");
        this._wsServer.close((err) => {
          if (err !== null && err !== undefined) {
            log("Failed to close websocket server");
            reject(err);
            return;
          }

          log("Websocket server closed");
          resolve();
        });
      }),
    ]);
  };
}
