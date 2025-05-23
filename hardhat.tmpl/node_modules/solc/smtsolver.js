"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
const command_exists_1 = require("command-exists");
const child_process_1 = require("child_process");
const fs = __importStar(require("fs"));
const tmp = __importStar(require("tmp"));
// Timeout in ms.
const timeout = 10000;
const potentialSolvers = [
    {
        name: 'z3',
        command: 'z3',
        params: '-smt2 rlimit=20000000 rewriter.pull_cheap_ite=true fp.spacer.q3.use_qgen=true fp.spacer.mbqi=false fp.spacer.ground_pobs=false'
    },
    {
        name: 'Eldarica',
        command: 'eld',
        params: '-horn -t:' + (timeout / 1000) // Eldarica takes timeout in seconds.
    },
    {
        name: 'cvc5',
        command: 'cvc5',
        params: '--lang=smt2 --tlimit=' + timeout
    }
];
const solvers = potentialSolvers.filter(solver => (0, command_exists_1.sync)(solver.command));
function solve(query, solver) {
    if (solver === undefined) {
        if (solvers.length === 0) {
            throw new Error('No SMT solver available. Assertion checking will not be performed.');
        }
        else {
            solver = solvers[0];
        }
    }
    const tmpFile = tmp.fileSync({ postfix: '.smt2' });
    fs.writeFileSync(tmpFile.name, query);
    let solverOutput;
    try {
        solverOutput = (0, child_process_1.execSync)(solver.command + ' ' + solver.params + ' ' + tmpFile.name, {
            encoding: 'utf8',
            maxBuffer: 1024 * 1024 * 1024,
            stdio: 'pipe',
            timeout: timeout // Enforce timeout on the process, since solvers can sometimes go around it.
        }).toString();
    }
    catch (e) {
        // execSync throws if the process times out or returns != 0.
        // The latter might happen with z3 if the query asks for a model
        // for an UNSAT formula. We can still use stdout.
        solverOutput = e.stdout.toString();
        if (!solverOutput.startsWith('sat') &&
            !solverOutput.startsWith('unsat') &&
            !solverOutput.startsWith('unknown') &&
            !solverOutput.startsWith('(error') && // Eldarica reports errors in an sexpr, for example: '(error "Failed to reconstruct array model")'
            !solverOutput.startsWith('error')) {
            throw new Error('Failed to solve SMT query. ' + e.toString());
        }
    }
    // Trigger early manual cleanup
    tmpFile.removeCallback();
    return solverOutput;
}
module.exports = {
    smtSolver: solve,
    availableSolvers: solvers
};
