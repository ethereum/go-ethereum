import { ErrorListener as AntlrErrorListener } from 'antlr4';
declare class ErrorListener extends AntlrErrorListener<any> {
    private _errors;
    constructor();
    syntaxError(recognizer: any, offendingSymbol: any, line: number, column: number, message: string): void;
    getErrors(): any[];
    hasErrors(): boolean;
}
export default ErrorListener;
