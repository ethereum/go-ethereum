import antlr4 from 'antlr4';
declare class ErrorListener extends antlr4.error.ErrorListener {
    private _errors;
    constructor();
    syntaxError(recognizer: any, offendingSymbol: any, line: number, column: number, message: string): void;
    getErrors(): any[];
    hasErrors(): boolean;
}
export default ErrorListener;
