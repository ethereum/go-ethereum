import type { Dispatcher } from "undici";

import fsExtra from "fs-extra";
import path from "path";

import { getHardhatVersion } from "./packageInfo";
import { shouldUseProxy } from "./proxy";

const TEMP_FILE_PREFIX = "tmp-";

function resolveTempFileName(filePath: string): string {
  const { dir, ext, name } = path.parse(filePath);

  return path.format({
    dir,
    ext,
    name: `${TEMP_FILE_PREFIX}${name}`,
  });
}

export async function download(
  url: string,
  filePath: string,
  timeoutMillis = 10000,
  extraHeaders: { [name: string]: string } = {}
) {
  const { getGlobalDispatcher, ProxyAgent, request } = await import("undici");

  let dispatcher: Dispatcher;
  if (process.env.http_proxy !== undefined && shouldUseProxy(url)) {
    dispatcher = new ProxyAgent(process.env.http_proxy);
  } else {
    dispatcher = getGlobalDispatcher();
  }

  const hardhatVersion = getHardhatVersion();

  // Fetch the url
  const response = await request(url, {
    dispatcher,
    headersTimeout: timeoutMillis,
    maxRedirections: 10,
    method: "GET",
    headers: {
      ...extraHeaders,
      "User-Agent": `hardhat ${hardhatVersion}`,
    },
  });

  if (response.statusCode >= 200 && response.statusCode <= 299) {
    const responseBody = Buffer.from(await response.body.arrayBuffer());
    const tmpFilePath = resolveTempFileName(filePath);
    await fsExtra.ensureDir(path.dirname(filePath));

    await fsExtra.writeFile(tmpFilePath, responseBody);
    return fsExtra.move(tmpFilePath, filePath, { overwrite: true });
  }
  // undici's response bodies must always be consumed to prevent leaks
  const text = await response.body.text();

  // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
  throw new Error(
    `Failed to download ${url} - ${response.statusCode} received. ${text}`
  );
}
