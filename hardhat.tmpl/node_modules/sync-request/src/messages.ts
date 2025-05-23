import {Response, HttpVerb} from 'then-request';
import {MessageOptions} from './Options';
export type Req = {
  m: HttpVerb;
  u: string;
  o?: MessageOptions;
};
export interface Res {
  s: Response['statusCode'];
  h: Response['headers'];
  b: Response['body'];
  u: Response['url'];
}
