#! /bin/env node

var fs = require('fs');
var validate = require('jsonschema').validate;
var readline = require('readline');
var process = require('process');

var schemaFile = process.argv[2];

var schema = '';
var testCode = '';
var success = true;
var numFiles = 0;
var numFailed = 0;
var numSucceeded = 0;
var fileNames = [];

var rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false
});

rl.on('line', function(fileName) {
    if (fileName == 'BlockchainTests/bcForgedTest/bcInvalidRLPTest.json') {
        return
    }
    fileNames.push(fileName);
});

rl.on('close', function() {
    schema = JSON.parse(fs.readFileSync(schemaFile));
    
    //sort file names alphabetically so that log output ordering is consistent
    fileNames.sort(function(a,b) {
        if(a<b) {
            return -1;
        } else {
            return 1;
        }

        return 0;
    });

    for (var i = 0; i < fileNames.length; i++) {
        try {
            testCode = JSON.parse(fs.readFileSync(fileNames[i]));
        } catch(e) {
            console.log('error on file:', fileNames[i])
            console.log(e);
            numFailed++;
            continue;
        }

        try {
            var x = validate(testCode, schema);

            if (x.errors.length > 0) {
                numFailed++;
                console.log(fileNames[i]+ ':\n');
                for (var j = 0; j < x.errors.length; j++) {
                    console.log('   ' + x.errors[j] + '\n')
                }
            } else {
                numSucceeded++;
            }
        } catch (e) {
            console.log(e);
            numFailed++;
        }
    }
    
    console.log("Valid: "+numSucceeded+"\n");
    console.log("Failed: "+numFailed+"\n");

    if(numFailed > 0) {
        process.exit(-1);
    }
});
