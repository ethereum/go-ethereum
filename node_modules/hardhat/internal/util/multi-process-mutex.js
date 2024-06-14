"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.MultiProcessMutex = void 0;
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
const debug_1 = __importDefault(require("debug"));
const node_fs_1 = __importDefault(require("node:fs"));
const node_path_1 = __importDefault(require("node:path"));
const node_os_1 = __importDefault(require("node:os"));
// Logic explanation: the fs.writeFile function, when used with the wx+ flag, performs an atomic operation to create a file.
// If multiple processes try to create the same file simultaneously, only one will succeed.
// This logic can be utilized to implement a mutex.
// ATTENTION: in the current implementation, there's still a risk of two processes running simultaneously.
// For example, if processA has locked the mutex and is running, processB will wait.
// During this wait, processB continuously checks the elapsed time since the mutex lock file was created.
// If an excessive amount of time has passed, processB will assume ownership of the mutex to avoid stale locks.
// However, there's a possibility that processB might take ownership because the mutex creation file is outdated, even though processA is still running
// For more info check the Nomic Notion page (internal link).
const log = (0, debug_1.default)("hardhat:util:multi-process-mutex");
const DEFAULT_MAX_MUTEX_LIFESPAN_IN_MS = 60000;
const MUTEX_LOOP_WAITING_TIME_IN_MS = 100;
class MultiProcessMutex {
    constructor(mutexName, maxMutexLifespanInMs) {
        log(`Creating mutex with name '${mutexName}'`);
        this._mutexFilePath = node_path_1.default.join(node_os_1.default.tmpdir(), `${mutexName}.txt`);
        this._mutexLifespanInMs =
            maxMutexLifespanInMs ?? DEFAULT_MAX_MUTEX_LIFESPAN_IN_MS;
    }
    async use(f) {
        log(`Starting mutex process with mutex file '${this._mutexFilePath}'`);
        while (true) {
            if (await this._tryToAcquireMutex()) {
                // Mutex has been acquired
                return this._executeFunctionAndReleaseMutex(f);
            }
            // Mutex not acquired
            if (this._isMutexFileTooOld()) {
                // If the mutex file is too old, it likely indicates a stale lock, so the file should be removed
                log(`Current mutex file is too old, removing it at path '${this._mutexFilePath}'`);
                this._deleteMutexFile();
            }
            else {
                // wait
                await this._waitMs();
            }
        }
    }
    async _tryToAcquireMutex() {
        try {
            // Create a file only if it does not exist
            node_fs_1.default.writeFileSync(this._mutexFilePath, "", { flag: "wx+" });
            return true;
        }
        catch (error) {
            if (error.code === "EEXIST") {
                // File already exists, so the mutex is already acquired
                return false;
            }
            throw error;
        }
    }
    async _executeFunctionAndReleaseMutex(f) {
        log(`Mutex acquired at path '${this._mutexFilePath}'`);
        try {
            const res = await f();
            // Release the mutex
            log(`Mutex released at path '${this._mutexFilePath}'`);
            this._deleteMutexFile();
            log(`Mutex released at path '${this._mutexFilePath}'`);
            return res;
        }
        catch (error) {
            // Catch any error to avoid stale locks.
            // Remove the mutex file and re-throw the error
            this._deleteMutexFile();
            throw error;
        }
    }
    _isMutexFileTooOld() {
        let fileStat;
        try {
            fileStat = node_fs_1.default.statSync(this._mutexFilePath);
        }
        catch (error) {
            if (error.code === "ENOENT") {
                // The file might have been deleted by another process while this function was trying to access it.
                return false;
            }
            throw error;
        }
        const now = new Date();
        const fileDate = new Date(fileStat.ctime);
        const diff = now.getTime() - fileDate.getTime();
        return diff > this._mutexLifespanInMs;
    }
    _deleteMutexFile() {
        try {
            log(`Deleting mutex file at path '${this._mutexFilePath}'`);
            node_fs_1.default.unlinkSync(this._mutexFilePath);
        }
        catch (error) {
            if (error.code === "ENOENT") {
                // The file might have been deleted by another process while this function was trying to access it.
                return;
            }
            throw error;
        }
    }
    async _waitMs() {
        return new Promise((resolve) => setTimeout(resolve, MUTEX_LOOP_WAITING_TIME_IN_MS));
    }
}
exports.MultiProcessMutex = MultiProcessMutex;
//# sourceMappingURL=multi-process-mutex.js.map