import type * as Undici from "undici";

export async function sendGetRequest(
  url: URL
): Promise<Undici.Dispatcher.ResponseData> {
  const { request } = await import("undici");
  const dispatcher = getDispatcher();

  return request(url, {
    dispatcher,
    method: "GET",
  });
}

export async function sendPostRequest(
  url: URL,
  body: string,
  headers: Record<string, string> = {}
): Promise<Undici.Dispatcher.ResponseData> {
  const { request } = await import("undici");
  const dispatcher = getDispatcher();

  return request(url, {
    dispatcher,
    method: "POST",
    headers,
    body,
  });
}

function getDispatcher(): Undici.Dispatcher {
  const { ProxyAgent, getGlobalDispatcher } =
    require("undici") as typeof Undici;
  if (process.env.http_proxy !== undefined) {
    return new ProxyAgent(process.env.http_proxy);
  }

  return getGlobalDispatcher();
}

export function isSuccessStatusCode(statusCode: number): boolean {
  return statusCode >= 200 && statusCode <= 299;
}
