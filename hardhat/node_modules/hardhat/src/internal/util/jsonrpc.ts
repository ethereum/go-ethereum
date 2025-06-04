import { HardhatError } from "../core/errors";
import { ERRORS } from "../core/errors-list";

export interface JsonRpcRequest {
  jsonrpc: string;
  method: string;
  params: any[];
  id: number | string;
}

export interface SuccessfulJsonRpcResponse {
  jsonrpc: string;
  id: number | string;
  result: any;
}

export interface FailedJsonRpcResponse {
  jsonrpc: string;
  id: number | string | null;
  error: {
    code: number;
    message: string;
    data?: any;
  };
}

export type JsonRpcResponse = SuccessfulJsonRpcResponse | FailedJsonRpcResponse;

export function parseJsonResponse(
  text: string
): JsonRpcResponse | JsonRpcResponse[] {
  try {
    const json = JSON.parse(text);

    const responses = Array.isArray(json) ? json : [json];
    for (const response of responses) {
      if (!isValidJsonResponse(response)) {
        // We are sending the proper error inside the catch part of the statement.
        // We just need to raise anything here.
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw new Error();
      }
    }

    return json;
  } catch {
    throw new HardhatError(ERRORS.NETWORK.INVALID_JSON_RESPONSE, {
      response: text,
    });
  }
}

export function isValidJsonRequest(payload: any): boolean {
  if (payload.jsonrpc !== "2.0") {
    return false;
  }

  if (typeof payload.id !== "number" && typeof payload.id !== "string") {
    return false;
  }

  if (typeof payload.method !== "string") {
    return false;
  }

  if (payload.params !== undefined && !Array.isArray(payload.params)) {
    return false;
  }

  return true;
}

export function isValidJsonResponse(payload: any) {
  if (payload.jsonrpc !== "2.0") {
    return false;
  }

  if (
    typeof payload.id !== "number" &&
    typeof payload.id !== "string" &&
    payload.id !== null
  ) {
    return false;
  }

  if (payload.id === null && payload.error === undefined) {
    return false;
  }

  if (payload.result === undefined && payload.error === undefined) {
    return false;
  }

  if (payload.error !== undefined) {
    if (typeof payload.error.code !== "number") {
      return false;
    }

    if (typeof payload.error.message !== "string") {
      return false;
    }
  }

  return true;
}

export function isSuccessfulJsonResponse(
  payload: JsonRpcResponse
): payload is SuccessfulJsonRpcResponse {
  return "result" in payload;
}
