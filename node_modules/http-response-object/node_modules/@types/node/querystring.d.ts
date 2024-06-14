declare module "querystring" {
    interface StringifyOptions {
        encodeURIComponent?: Function;
    }

    interface ParseOptions {
        maxKeys?: number;
        decodeURIComponent?: Function;
    }

    interface ParsedUrlQuery { [key: string]: string | string[]; }

    function stringify(obj?: {}, sep?: string, eq?: string, options?: StringifyOptions): string;
    function parse(str: string, sep?: string, eq?: string, options?: ParseOptions): ParsedUrlQuery;
    function escape(str: string): string;
    function unescape(str: string): string;
}
