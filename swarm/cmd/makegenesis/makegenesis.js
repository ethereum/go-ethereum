var Account = require('ethereumjs-account');
var async = require('async');
var fs = require('fs');
var rlp = require('rlp');
var solc = require('solc');
var Transaction = require('ethereumjs-tx');
var Trie = require('merkle-patricia-tree');
var utils = require('ethereumjs-util');
var VM = require('ethereumjs-vm');

var stateTrie = new Trie();
var vm = new VM(stateTrie);

var privatekey = utils.sha3("swarm");
var accountAddress = utils.privateToAddress(privatekey);

var accounts = {
    "0000000000000000000000000000000000000001": "0x1",
    "0000000000000000000000000000000000000002": "0x1",
    "0000000000000000000000000000000000000003": "0x1",
    "0000000000000000000000000000000000000004": "0x1",
};
accounts[accountAddress.toString('hex')] = "0x200000000000000";

var source = fs.readFileSync('../../services/ens/contract/ens.sol').toString();
var compiled = solc.compile(source, 1)
var deployer = "0x" + compiled.contracts['DeployENS'].bytecode;

var transactions = [
    {
        nonce: '0x00',
        gasPrice: '0x4a817c800',
        gasLimit: '0x3d0900',
        value: '0x00',
        data: deployer
    }
];

function createAccounts(cb) {
    async.each(Object.keys(accounts), function(addr, next) {
        var account = new Account();
        account.balance = accounts[addr];
        stateTrie.put(new Buffer(addr, 'hex'), account.serialize(), next);
    }, cb);
}

function runTx(cb) {
    async.each(transactions, function(txdata, next) {
        var tx = new Transaction(txdata);
        tx.sign(privatekey);
        vm.runTx({tx: tx}, next);
    }, cb);
}

function dumpState(cb) {
    var stream = stateTrie.createReadStream();
    
    var accountList = [];
    stream.on('data', function(data) {
        accountList.push(data);
    });
    stream.on('end', function(err) {
        var accountData = {};
        async.each(accountList, function(data, next) {
            var storage = {};

            var account = new Account(data.value);
            var storageTrie = stateTrie.copy();
            storageTrie.root = account.stateRoot;

            var storageStream = storageTrie.createReadStream();
            storageStream.on('data', function(data) {
                storage[data.key.toString('hex')] = rlp.decode(data.value).toString('hex')
            });
            storageStream.on('end', function(err) {
                account.getCode(stateTrie, function(err, code) {
                    var address = data.key.toString('hex');
                    accountData[address] = {
                        code: code.toString('hex'),
                        storage: storage,
                        balance: account.balance.toString('hex')
                    };
                    next(err);
                });
            })
        }, function(err) {
            console.log(JSON.stringify({
                nonce: '0x42',
                gasLimit: '0x47e7c4',
                difficulty: '0x20000',
                alloc: accountData
            }, null, 4));
        });
    });
}

async.series([
    createAccounts,
    runTx,
    dumpState
], function(err) {
    console.log("Error: " + err);
});
