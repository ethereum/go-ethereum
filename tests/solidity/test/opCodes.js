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
   it('should add a to-do note successfully with a short text of 20 letters', async () => {
     await contractInstance.test()
    //  await contractInstance.test_revert()
    //  await contractInstance.test_invalid()
     await contractInstance.test_stop()

   })
//   it('should mark one of your to-dos as completed', async () => {
//    await contractInstance.addTodo('example')
//    await contractInstance.markTodoAsCompleted(0)
//    const lastTodoAdded = await contractInstance.todos(accounts[0], 0)
//    const isTodoCompleted = lastTodoAdded[3] // 3 is the bool isCompleted value of the todo note
//    assert(isTodoCompleted, 'The todo should be true as completed')
// })
})
