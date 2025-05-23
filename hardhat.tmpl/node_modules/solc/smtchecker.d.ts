declare function handleSMTQueries(inputJSON: any, outputJSON: any, solverFunction: any, solver?: any): any;
declare function smtCallback(solverFunction: any, solver?: any): (query: any) => {
    contents: any;
    error?: undefined;
} | {
    error: any;
    contents?: undefined;
};
declare const _default: {
    handleSMTQueries: typeof handleSMTQueries;
    smtCallback: typeof smtCallback;
};
export = _default;
