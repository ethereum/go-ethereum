"use strict";
// This function checks the standard JSON output for auxiliaryInputRequested,
// where smtlib2queries represent the queries created by the SMTChecker.
// The function runs an SMT solver on each query and adjusts the input for
// another run.
// Returns null if no solving is requested.
function handleSMTQueries(inputJSON, outputJSON, solverFunction, solver) {
    const auxInputReq = outputJSON.auxiliaryInputRequested;
    if (!auxInputReq) {
        return null;
    }
    const queries = auxInputReq.smtlib2queries;
    if (!queries || Object.keys(queries).length === 0) {
        return null;
    }
    const responses = {};
    for (const query in queries) {
        responses[query] = solverFunction(queries[query], solver);
    }
    // Note: all existing solved queries are replaced.
    // This assumes that all necessary queries are quested above.
    inputJSON.auxiliaryInput = { smtlib2responses: responses };
    return inputJSON;
}
function smtCallback(solverFunction, solver) {
    return function (query) {
        try {
            const result = solverFunction(query, solver);
            return { contents: result };
        }
        catch (err) {
            return { error: err };
        }
    };
}
module.exports = {
    handleSMTQueries,
    smtCallback
};
