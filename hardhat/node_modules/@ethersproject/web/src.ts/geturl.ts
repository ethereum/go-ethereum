"use strict";

import http from "http";
import https from "https";
import { gunzipSync } from "zlib";
import { parse } from "url"

import { arrayify, concat } from "@ethersproject/bytes";
import { shallowCopy } from "@ethersproject/properties";

import type { GetUrlResponse, Options } from "./types";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export { GetUrlResponse, Options };

function getResponse(request: http.ClientRequest): Promise<GetUrlResponse> {
    return new Promise((resolve, reject) => {
        request.once("response", (resp: http.IncomingMessage) => {
            const response: GetUrlResponse = {
                statusCode: resp.statusCode,
                statusMessage: resp.statusMessage,
                headers: Object.keys(resp.headers).reduce((accum, name) => {
                    let value = resp.headers[name];
                    if (Array.isArray(value)) {
                        value = value.join(", ");
                    }
                    accum[name] = value;
                    return accum;
                }, <{ [ name: string ]: string }>{ }),
                body: null
            };
            //resp.setEncoding("utf8");

            resp.on("data", (chunk: Uint8Array) => {
                if (response.body == null) { response.body = new Uint8Array(0); }
                response.body = concat([ response.body, chunk ]);
            });

            resp.on("end", () => {
                if (response.headers["content-encoding"] === "gzip") {
                    //const size = response.body.length;
                    response.body = arrayify(gunzipSync(response.body));
                    //console.log("Delta:", response.body.length - size, Buffer.from(response.body).toString());
                }
                resolve(response);
            });

            resp.on("error", (error) => {
                /* istanbul ignore next */
                (<any>error).response = response;
                reject(error);
            });
        });

        request.on("error", (error) => { reject(error); });
    });
}

// The URL.parse uses null instead of the empty string
function nonnull(value: string): string {
    if (value == null) { return ""; }
    return value;
}

export async function getUrl(href: string, options?: Options): Promise<GetUrlResponse> {
    if (options == null) { options = { }; }

    // @TODO: Once we drop support for node 8, we can pass the href
    //        directly into request and skip adding the components
    //        to this request object
    const url = parse(href);

    const request = {
        protocol: nonnull(url.protocol),
        hostname: nonnull(url.hostname),
        port: nonnull(url.port),
        path: (nonnull(url.pathname) + nonnull(url.search)),

        method: (options.method || "GET"),
        headers: shallowCopy(options.headers || { }),
    };

    if (options.allowGzip) {
        request.headers["accept-encoding"] = "gzip";
    }

    let req: http.ClientRequest = null;
    switch (nonnull(url.protocol)) {
        case "http:":
            req = http.request(request);
            break;
        case "https:":
            req = https.request(request);
            break;
        default:
            /* istanbul ignore next */
            logger.throwError(`unsupported protocol ${ url.protocol }`, Logger.errors.UNSUPPORTED_OPERATION, {
                protocol: url.protocol,
                operation: "request"
            });
    }

    if (options.body) {
        req.write(Buffer.from(options.body));
    }
    req.end();

    const response = await getResponse(req);
    return response;
}

