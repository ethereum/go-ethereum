/** JSDoc */
export interface StackFrame {
    filename?: string;
    function?: string;
    module?: string;
    platform?: string;
    lineno?: number;
    colno?: number;
    abs_path?: string;
    context_line?: string;
    pre_context?: string[];
    post_context?: string[];
    in_app?: boolean;
    vars?: {
        [key: string]: any;
    };
}
//# sourceMappingURL=stackframe.d.ts.map