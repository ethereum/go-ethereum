export declare function printLine(line: string): void;
export declare function replaceLastLine(newLine: string): void;
export interface LoggerConfig {
    enabled: boolean;
    printLineFn?: (line: string) => void;
    replaceLastLineFn?: (line: string) => void;
}
//# sourceMappingURL=logger.d.ts.map