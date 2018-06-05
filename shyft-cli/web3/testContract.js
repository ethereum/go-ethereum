var testContract = web3.eth.contract([{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"live","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_bool","type":"bool"}],"name":"Live","type":"event"}]);
var test = testContract.new(
    {
        from: web3.eth.accounts[0],
        data: '0x6060604052341561000f57600080fd5b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506101698061005e6000396000f30060606040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680638da5cb5b14610051578063957aa58c146100a6575b600080fd5b341561005c57600080fd5b6100646100d3565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156100b157600080fd5b6100b96100f8565b604051808215151515815260200191505060405180910390f35b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60007f63f89985c88a92552c6e2ce4d4e46e9fbb7a06168c4536e662261decae02f9e76001604051808215151515815260200191505060405180910390a160019050905600a165627a7a7230582091089d445770c1c9bb2f0f86f6c0b8552ef59af41930e1f7ecfac34326b20e590029',
        gas: '4700000'
    }, function (e, contract){
        if (e) {
            console.log("Error: ", e)
        }
        console.log("TX Hash:", contract.transactionHash)
        if (typeof contract.address !== 'undefined') {
            console.log('Contract address: ' + contract.address)
            console.log('TransactionHash: ' + contract.transactionHash);
            interact(contract.address);
        }
    });

function waitBlock(callback) {
    function innerWaitBlock() {

        var receipt = web3.eth.getTransactionReceipt(test.transactionHash);
        if (receipt && receipt.contractAddress) {
            callback(receipt);
        } else {
            // console.log("Waiting a mined block to include your contract... currently in block " + web3.eth.blockNumber);
            setTimeout(innerWaitBlock(), 4000);
        }
    }
    innerWaitBlock();
}

waitBlock(function (receipt) {
    // do stuff here now that the contract has been deployed
    console.log("[Receipt] Contract Address: ", receipt.contract.address)
});

function interact(addr) {
    var contract = testContract.at(addr);
    contract.live({
        from: web3.eth.accounts[0],
        gas: '4700000'
    }, function (e, res) {
        if (e) {
            console.log(e)
        }
        console.log('Live TxHash: ', res)
    });
    contract.live.call({
        from: web3.eth.accounts[0],
        gas: '4700000'
    }, function (e, res) {
        if (e) {
            console.log(e)
        }
        console.log('Live Response: ', res)
    })
}