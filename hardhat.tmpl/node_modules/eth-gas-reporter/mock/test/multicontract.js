const MultiContractFileA = artifacts.require('MultiContractFileA');
const MultiContractFileB = artifacts.require('MultiContractFileB');

contract('MultiContractFiles', accounts => {
  let a
  let b

  beforeEach(async function () {
    a = await MultiContractFileA.new()
    b = await MultiContractFileB.new()
  })

  it('a and b', async function(){
    await a.hello();
    await b.goodbye();
  })
});