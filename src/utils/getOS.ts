import { OS } from '../types';

// slice to get the last part of the url that includes the os
export const getOS = (url: string): OS =>
  url?.slice(46).includes('darwin')
    ? 'darwin'
    : url?.slice(46).includes('linux')
    ? 'linux'
    : url?.slice(46).includes('windows')
    ? 'windows'
    : 'mobile';
