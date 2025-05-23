const ethers = require("ethers");

/**
 * Example of a method that resolves the contract names of method calls routed through 
 * an EtherRouter style contract. This function gets bound to the `this` property of 
 * eth-gas-reporter's ProxyResolver class and inherits its resources including 
 * helpers to match methods to contracts and a way of making synchronous calls to the client.
 *
 * Helper methods of this type receive a web3 transaction object representing a tx the reporter
 * could not deterministically associate with any contract. They rely on your knowledge
 * of a proxy contract's API to derive the correct contract name.
 *
 * Returns contract name matching the resolved address.
 * @param  {Object} transaction result of web3.eth.getTransaction
 * @return {String}             contract name
 */
function etherRouter(transaction) {
  let contractAddress;
  let contractName;

  try {
    const ABI = ["function resolver()", "function lookup(bytes4 sig)"];
    const iface = new ethers.utils.Interface(ABI);
    
    // The tx passed to this method had input data which didn't map to any methods on
    // the contract it was sent to. It's possible the tx's `to` address points to  
    // an EtherRouter contract which is designed to forward calls. We'll grab the 
    // method signature and ask the router if it knows who the intended recipient is.
    const signature = transaction.input.slice(0, 10);

    // EtherRouter has a public state variable called `resolver()` which stores the 
    // address of a contract which maps method signatures to their parent contracts.
    // Lets fetch it ....
    const resolverAddress = this.sync.call(
      {
        to: transaction.to,
        data: iface.functions.resolver.encode([])
      },
      transaction.blockNumber
    );

    // Now we'll call the Resolver's `lookup(sig)` method to get the address of the contract 
    // our tx was actually getting forwarded to. 
    contractAddress = this.sync.call(
      {
        to: ethers.utils.hexStripZeros(resolverAddress),
        data: iface.functions.lookup.encode([signature])
      },
      transaction.blockNumber
    );
  // Don't forget this is all a bit speculative...
  } catch (err) {
    this.unresolvedCalls++;
    return;
  }

  // With the correct address, we can use the ProxyResolver class's 
  // data.getNameByAddress and/or resolveByDeployedBytecode methods 
  // (both are available in this scope, bound to `this`) to derive 
  // the target contract's name.
  if (contractAddress) {
    contractAddress = ethers.utils.hexStripZeros(contractAddress);
    contractName = this.data.getNameByAddress(contractAddress);

    // Try to resolve by deployedBytecode
    if (contractName) return contractName;
    else return this.resolveByDeployedBytecode(contractAddress);
  }
}

module.exports = etherRouter;
