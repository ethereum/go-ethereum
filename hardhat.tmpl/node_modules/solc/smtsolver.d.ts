declare function solve(query: any, solver: any): any;
declare const _default: {
    smtSolver: typeof solve;
    availableSolvers: {
        name: string;
        command: string;
        params: string;
    }[];
};
export = _default;
