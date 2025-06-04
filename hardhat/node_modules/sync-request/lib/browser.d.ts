/// <reference types="node" />
import { URL } from 'url';
import { HttpVerb, Response } from 'then-request';
import { Options } from './Options';
declare const fd: any;
export { fd as FormData };
export default function doRequest(method: HttpVerb, url: string | URL, options?: Options): Response;
