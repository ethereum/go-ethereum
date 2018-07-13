var Transfers = artifacts.require("Transfers");
var Transfers2 = artifacts.require("Transfers2");

contract('Transfers', function(accounts) {
    var addressA = web3.eth.accounts[0];
    var addressB = web3.eth.accounts[1];
    console.log("Addr A:", addressA)
    console.log("Addr B:", addressB)

    var transfers;
    web3.eth.defaultAccount = web3.eth.accounts[0];

    it("should initialize", async() => {
    	transfers = await Transfers.new();
    	transfers2 = await Transfers2.new();
    	console.log("Transfer addr1", transfers.address)
        console.log("Transfer addr2", transfers2.address)

        assert(transfers !== undefined, "");
    	assert(transfers2 !== undefined, "");
    })

    it("should deposit", async() => {
    	let hash = await transfers.deposit({from: addressA, value: web3.toWei(4, "ether")});
    	let bal = await transfers.deposit.call({from: addressA});
        console.log("\t\t[DEPOSIT TX]", hash.tx)
		//console.log(bal);
    })

    it("should withdraw", async() => {
    	let prevB = await web3.eth.getBalance(addressB);
    	let hash = await transfers.withdraw(addressB, web3.toWei(1, "ether"), {from: addressA});
    	let postB = await web3.eth.getBalance(addressB);
        console.log("\t\t[WITHDRAW TX]", hash.tx)
    	//assert(postB - prevB == val, "");
    })

    it("should transfer through other contract", async() => {
    	let val = 10;
    	let prevB = await web3.eth.getBalance(addressB);

    	let hash = await transfers.transfer(addressB, web3.toWei(1, "ether"), {from: addressA});
    	let postB = await web3.eth.getBalance(addressB);
        console.log("\t\t[TRANSFER THROUGH TX]", hash.tx)
    	// assert(web3.fromWei(prevB,'wei') - web3.fromWei(postB,'wei') == val, "");
    })
})