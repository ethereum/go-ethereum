import type * as Undici from "undici";

import { EventEmitter } from "events";

import { EIP1193Provider, RequestArguments } from "../../../types";
import {
  HARDHAT_NETWORK_RESET_EVENT,
  HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT,
} from "../../constants";
import {
  FailedJsonRpcResponse,
  JsonRpcRequest,
  JsonRpcResponse,
  parseJsonResponse,
  SuccessfulJsonRpcResponse,
} from "../../util/jsonrpc";
import { getHardhatVersion } from "../../util/packageInfo";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
import { shouldUseProxy } from "../../util/proxy";

import { ProviderError } from "./errors";

export function isErrorResponse(
  response: any
): response is FailedJsonRpcResponse {
  return typeof response.error !== "undefined";
}

const MAX_RETRIES = 6;
const MAX_RETRY_AWAIT_SECONDS = 5;

const TOO_MANY_REQUEST_STATUS = 429;

const hardhatVersion = getHardhatVersion();

export class HttpProvider extends EventEmitter implements EIP1193Provider {
  private _nextRequestId = 1;
  private _dispatcher: Undici.Dispatcher;
  private _path: string;
  private _authHeader: string | undefined;

  constructor(
    private readonly _url: string,
    private readonly _networkName: string,
    private readonly _extraHeaders: { [name: string]: string } = {},
    private readonly _timeout = 20000,
    client: Undici.Dispatcher | undefined = undefined
  ) {
    super();

    const { Pool, ProxyAgent } = require("undici") as typeof Undici;

    if (this._url.trim().length === 0) {
      throw new HardhatError(ERRORS.NETWORK.EMPTY_URL, {
        value: this._url,
      });
    }

    const url = new URL(this._url);
    this._path = url.pathname;
    this._authHeader =
      url.username === ""
        ? undefined
        : `Basic ${Buffer.from(
            `${url.username}:${url.password}`,
            "utf-8"
          ).toString("base64")}`;
    try {
      this._dispatcher = client ?? new Pool(url.origin);

      if (process.env.http_proxy !== undefined && shouldUseProxy(url.origin)) {
        this._dispatcher = new ProxyAgent(process.env.http_proxy);
      }
    } catch (e) {
      if (e instanceof TypeError && e.message === "Invalid URL") {
        e.message += ` ${url.origin}`;
      }
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw e;
    }
  }

  public get url(): string {
    return this._url;
  }

  public async request(args: RequestArguments): Promise<unknown> {
    const jsonRpcRequest = this._getJsonRpcRequest(
      args.method,
      args.params as any[]
    );
    const jsonRpcResponse = await this._fetchJsonRpcResponse(jsonRpcRequest);

    if (isErrorResponse(jsonRpcResponse)) {
      const error = new ProviderError(
        jsonRpcResponse.error.message,
        jsonRpcResponse.error.code
      );
      error.data = jsonRpcResponse.error.data;
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }

    if (args.method === "hardhat_reset") {
      this.emit(HARDHAT_NETWORK_RESET_EVENT);
    }
    if (args.method === "evm_revert") {
      this.emit(HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT);
    }

    return jsonRpcResponse.result;
  }

  /**
   * Sends a batch of requests. Fails if any of them fails.
   */
  public async sendBatch(
    batch: Array<{ method: string; params: any[] }>
  ): Promise<any[]> {
    // We create the errors here to capture the stack traces at this point,
    // the async call that follows would probably loose of the stack trace
    const stackSavingError = new ProviderError("HttpProviderError", -1);

    // we need this to sort the responses
    const idToIndexMap: Record<string, number> = {};

    const requests = batch.map((r, i) => {
      const jsonRpcRequest = this._getJsonRpcRequest(r.method, r.params);
      idToIndexMap[jsonRpcRequest.id] = i;
      return jsonRpcRequest;
    });

    const jsonRpcResponses = await this._fetchJsonRpcResponse(requests);

    for (const response of jsonRpcResponses) {
      if (isErrorResponse(response)) {
        const error = new ProviderError(
          response.error.message,
          response.error.code,
          stackSavingError
        );
        error.data = response.error.data;
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
      }
    }

    // We already know that it has this type, but TS can't infer it.
    const responses = jsonRpcResponses as SuccessfulJsonRpcResponse[];

    // we use the id to sort the responses so that they match the order of the requests
    const sortedResponses = responses
      .map(
        (response) =>
          [idToIndexMap[response.id], response.result] as [number, any]
      )
      .sort(([indexA], [indexB]) => indexA - indexB)
      .map(([, result]) => result);

    return sortedResponses;
  }

  private async _fetchJsonRpcResponse(
    request: JsonRpcRequest,
    retryNumber?: number
  ): Promise<JsonRpcResponse>;
  private async _fetchJsonRpcResponse(
    request: JsonRpcRequest[],
    retryNumber?: number
  ): Promise<JsonRpcResponse[]>;
  private async _fetchJsonRpcResponse(
    request: JsonRpcRequest | JsonRpcRequest[],
    retryNumber?: number
  ): Promise<JsonRpcResponse | JsonRpcResponse[]>;
  private async _fetchJsonRpcResponse(
    request: JsonRpcRequest | JsonRpcRequest[],
    retryNumber = 0
  ): Promise<JsonRpcResponse | JsonRpcResponse[]> {
    const { request: sendRequest } = await import("undici");
    const url = new URL(this._url);

    const headers: { [name: string]: string } = {
      "Content-Type": "application/json",
      "User-Agent": `hardhat ${hardhatVersion}`,
      ...this._extraHeaders,
    };

    if (this._authHeader !== undefined) {
      headers.Authorization = this._authHeader;
    }

    try {
      const response = await sendRequest(url, {
        dispatcher: this._dispatcher,
        method: "POST",
        body: JSON.stringify(request),
        maxRedirections: 10,
        headersTimeout:
          process.env.DO_NOT_SET_THIS_ENV_VAR____IS_HARDHAT_CI !== undefined
            ? 0
            : this._timeout,
        headers,
      });

      if (this._isRateLimitResponse(response)) {
        // "The Fetch Standard allows users to skip consuming the response body
        // by relying on garbage collection to release connection resources.
        // Undici does not do the same. Therefore, it is important to always
        // either consume or cancel the response body."
        // https://undici.nodejs.org/#/?id=garbage-collection
        // It's not clear how to "cancel", so we'll just consume:
        await response.body.text();
        const seconds = this._getRetryAfterSeconds(response, retryNumber);
        if (seconds !== undefined && this._shouldRetry(retryNumber, seconds)) {
          return await this._retry(request, seconds, retryNumber);
        }

        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new ProviderError(
          `Too Many Requests error received from ${url.hostname}`,
          -32005 // Limit exceeded according to EIP1474
        );
      }

      return parseJsonResponse(await response.body.text());
    } catch (error: any) {
      if (error.code === "ECONNREFUSED") {
        throw new HardhatError(
          ERRORS.NETWORK.NODE_IS_NOT_RUNNING,
          { network: this._networkName },
          error
        );
      }

      if (error.type === "request-timeout") {
        throw new HardhatError(ERRORS.NETWORK.NETWORK_TIMEOUT, {}, error);
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }
  }

  private async _retry(
    request: JsonRpcRequest | JsonRpcRequest[],
    seconds: number,
    retryNumber: number
  ) {
    await new Promise((resolve) => setTimeout(resolve, 1000 * seconds));
    return this._fetchJsonRpcResponse(request, retryNumber + 1);
  }

  private _getJsonRpcRequest(
    method: string,
    params: any[] = []
  ): JsonRpcRequest {
    return {
      jsonrpc: "2.0",
      method,
      params,
      id: this._nextRequestId++,
    };
  }

  private _shouldRetry(retryNumber: number, retryAfterSeconds: number) {
    if (retryNumber > MAX_RETRIES) {
      return false;
    }

    if (retryAfterSeconds > MAX_RETRY_AWAIT_SECONDS) {
      return false;
    }

    return true;
  }

  private _isRateLimitResponse(response: Undici.Dispatcher.ResponseData) {
    return response.statusCode === TOO_MANY_REQUEST_STATUS;
  }

  private _getRetryAfterSeconds(
    response: Undici.Dispatcher.ResponseData,
    retryNumber: number
  ): number | undefined {
    const header = response.headers["retry-after"];

    if (header === undefined || header === null || Array.isArray(header)) {
      // if the response doesn't have a retry-after header, we do
      // an exponential backoff
      return Math.min(2 ** retryNumber, MAX_RETRY_AWAIT_SECONDS);
    }

    const parsed = parseInt(header, 10);
    if (isNaN(parsed)) {
      return undefined;
    }

    return parsed;
  }
}
