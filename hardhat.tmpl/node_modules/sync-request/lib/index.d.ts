/// <reference types="node" />
import { HttpVerb, Response } from 'then-request';
import { URL } from 'url';
import { FormData } from './FormData';
import { Options } from './Options';
export { HttpVerb, Response, Options };
export { FormData };
export default function request(method: HttpVerb, url: string | URL, options?: Options): Response;
