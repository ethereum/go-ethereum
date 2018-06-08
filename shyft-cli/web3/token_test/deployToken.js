var Web3 = require('web3')
var web3 = new Web3(new Web3.providers.HttpProvider("http://127.0.0.1:8545"));

var initialSupply = 1000000000000000
var mytokenContract = web3.eth.contract([{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"_bool","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"initialSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]);
var mytoken = mytokenContract.new(
    initialSupply,
    {
        from: web3.eth.accounts[0],
        data: '0x6060604052341561000f57600080fd5b60405160208061032f83398101604052808051906020019091905050806000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550506102b18061007e6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806370a0823114610048578063a9059cbb1461009557600080fd5b341561005357600080fd5b61007f600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506100ef565b6040518082815260200191505060405180910390f35b34156100a057600080fd5b6100d5600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035906020019091905050610107565b604051808215151515815260200191505060405180910390f35b60006020528060005260406000206000915090505481565b6000816000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541015151561015657600080fd5b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054826000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205401101515156101e357600080fd5b816000803373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282540392505081905550816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254019250508190555060019050929150505600a165627a7a723058207dba9cbbfe34ea34b40caa5e6d2b53f9194dafa2420207342bebe3a5c949840c0029',
        gas: '4700000'
    }, function (e, contract) {
        if (e) console.log("err1", e);
        if (typeof contract.address !== 'undefined') {
            console.log('Contract mined! address: ' + contract.address + ' transactionHash: ' + contract.transactionHash);
            var a = mytokenContract.at(contract.address);
            a.transfer.sendTransaction(web3.eth.accounts[0], 5000000, {from: web3.eth.accounts[0]}, function (err, res) {
                if (err) console.log("err", err);
                console.log(res)
            })
        }
    });