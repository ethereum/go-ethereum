const TodoList = artifacts.require('./OpCodes.sol')
const assert = require('assert')
let contractInstance
const Web3 = require('web3');
const web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:8545'));
// const web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:9545'));

contract('OpCodes', (accounts) => {
   beforeEach(async () => {
      contractInstance = await TodoList.deployed()
   })
   it('Should run without errors the majorit of opcodes', async () => {
     await contractInstance.test()
     await contractInstance.test_stop()

   })

   it('Should throw invalid op code', async () => {
    try{
      await contractInstance.test_invalid()
    }
    catch(error) {
      console.error(error);
    }
   })

   it('Should revert', async () => {
    try{
      await contractInstance.test_revert()    }
    catch(error) {
      console.error(error);
    }
   })
})
