import { HttpVerb } from 'http-basic';
import { Options } from './Options';
import { ResponsePromise } from './ResponsePromise';
declare type RequestFn = (method: HttpVerb, url: string, options?: Options) => ResponsePromise;
export { RequestFn };
