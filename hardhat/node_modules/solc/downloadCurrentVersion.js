#!/usr/bin/env node
"use strict";
// This is used to download the correct binary version
// as part of the prepublish step.
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const fs = __importStar(require("fs"));
const follow_redirects_1 = require("follow-redirects");
const memorystream_1 = __importDefault(require("memorystream"));
const js_sha3_1 = require("js-sha3");
const pkg = require('./package.json');
function getVersionList(cb) {
    console.log('Retrieving available version list...');
    const mem = new memorystream_1.default(null, { readable: false });
    follow_redirects_1.https.get('https://binaries.soliditylang.org/bin/list.json', function (response) {
        if (response.statusCode !== 200) {
            console.log('Error downloading file: ' + response.statusCode);
            process.exit(1);
        }
        response.pipe(mem);
        response.on('end', function () {
            cb(mem.toString());
        });
    });
}
function downloadBinary(outputName, version, expectedHash) {
    console.log('Downloading version', version);
    // Remove if existing
    if (fs.existsSync(outputName)) {
        fs.unlinkSync(outputName);
    }
    process.on('SIGINT', function () {
        console.log('Interrupted, removing file.');
        fs.unlinkSync(outputName);
        process.exit(1);
    });
    const file = fs.createWriteStream(outputName, { encoding: 'binary' });
    follow_redirects_1.https.get('https://binaries.soliditylang.org/bin/' + version, function (response) {
        if (response.statusCode !== 200) {
            console.log('Error downloading file: ' + response.statusCode);
            process.exit(1);
        }
        response.pipe(file);
        file.on('finish', function () {
            file.close(function () {
                const hash = '0x' + (0, js_sha3_1.keccak256)(fs.readFileSync(outputName, { encoding: 'binary' }));
                if (expectedHash !== hash) {
                    console.log('Hash mismatch: ' + expectedHash + ' vs ' + hash);
                    process.exit(1);
                }
                console.log('Done.');
            });
        });
    });
}
console.log('Downloading correct solidity binary...');
getVersionList(function (list) {
    list = JSON.parse(list);
    const wanted = pkg.version.match(/^(\d+\.\d+\.\d+)$/)[1];
    const releaseFileName = list.releases[wanted];
    const expectedFile = list.builds.filter(function (entry) { return entry.path === releaseFileName; })[0];
    if (!expectedFile) {
        console.log('Version list is invalid or corrupted?');
        process.exit(1);
    }
    const expectedHash = expectedFile.keccak256;
    downloadBinary('soljson.js', releaseFileName, expectedHash);
});
