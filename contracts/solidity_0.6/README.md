#  What’s new in Solidity 0.6 ?

[Solidity 0.6](https://solidity.readthedocs.io/en/v0.6.2/060-breaking-changes.html) was recently released, bringing a few syntactical changes and non-backward compatible changes.

Let's take a look at some of the major changes:

- Function Overriding
- Abstract Contracts
- Try / Catch
- Receive Ether / Fallback Function split
- Array Resizing
- State variable shadowing

### Function Overriding
In previous Solidity versions, there was no explicit way to know which functions an inheriting contract should override. Solidity 0.6 brings with it an improvement that makes it clear which functions an inheriting contract can override.

An inheriting contract can override its base contract function behavior only if they are marked as virtual. The overriding function must then use the override keyword in the function header as shown below.

```
pragma solidity ^0.6.0;
contract Base1
{
    function foo() virtual public {}
}
contract Base2
{
    function foo() virtual public {}
}
contract Inherited is Base1, Base2
{
    // Derives from multiple bases defining foo(), so we must explicitly
    // override it
    function foo() public override(Base1, Base2) {}
}
```

With this improvement no more checking OpenZeppelin docs as to which functions can be overridden as the Solidity compiler itself can let you know when an inheriting contract overrides a no `virtual` function.

### Abstract Contracts
A new keyword abstract was introduced in Solidity 0.6. It is used to mark contracts that do not implement all its functions.

Contracts are to be marked as abstractwhen at least one of their functions is not implemented. Functions without implementation outside an interface have to be marked virtual. Contracts can also be marked as abstract even though all functions are implemented.

```
pragma solidity ^0.6.0;
abstract contract Feline {
    function utterance() public virtual returns (bytes32);
}
contract Cat is Feline {
    function utterance() public override returns (bytes32) {
        return "miaow";
     }
}

```

### Try / Catch
Solidity 0.6 introduces a try/catch statement for handling failed contract call errors. It allows you to react to failed external calls.

A failure in an external contract call can be caught using a try/catch statement, as follows:

```pragma solidity ^0.6.0;
interface DataFeed { 
      function getData(address token) external returns (uint value); 
}
contract FeedConsumer {
    DataFeed feed;
    uint errorCount;
    function rate(address token) public returns (uint value, bool success) {
        // Permanently disable the mechanism if there are
        // more than 10 errors.
        require(errorCount < 10);
        try feed.getData(token) returns (uint v) {
            return (v, true);
        } catch Error(string memory /*reason*/) {
            // This is executed in case
            // revert was called inside getData
            // and a reason string was provided.
            errorCount++;
            return (0, false);
        } catch (bytes memory /*lowLevelData*/) {
            // This is executed in case revert() was used
            // or there was a failing assertion, division
            // by zero, etc. inside getData.
            errorCount++;
            return (0, false);
        }
    }
}
```

Solidity supports different kinds of catch blocks depending on the type of error. If the error was caused by `revert("reasonString") or require(false, "reasonString")` (or an internal error that causes such an exception), then the catch clause of the type `catch Error(string memory reason)` will be executed.

### Receive Ether / Fallback Function Split
#### Receive Ether
To enable a contract to receive Ether, it must implement `the receive()` function. A contract can have at most one receive function, declared using `receive() external payable { ... }` (without the `function` keyword) as shown in the example below:

```
pragma solidity ^0.6.0;
// This contract keeps all Ether sent to it with no way
// to get it back.
contract Sink {
    event Received(address, uint);
    receive() external payable {
        emit Received(msg.sender, msg.value);
    }
}
```

#### Fallback Function
A contract can have at most one fallback function, declared using `fallback () external [payable]` (without the `function` keyword). This function cannot have arguments, cannot return anything and must have external visibility. It is executed on a call to the contract if none of the other functions match the given function signature, or if no data was supplied at all and there is no [receive Ether function](https://solidity.readthedocs.io/en/v0.6.2/contracts.html#receive-ether-function). The fallback function always receives data, but in order to also receive Ether it must be marked payable.

```
pragma solidity >0.6.1 <0.7.0;
contract Test {
    // This function is called for all messages sent to
    // this contract (there is no other function).
    // Sending Ether to this contract will cause an exception,
    // because the fallback function does not have the `payable`
    // modifier.
    fallback() external { x = 1; }
    uint x;
}
contract TestPayable {
    // This function is called for all messages sent to
    // this contract, except plain Ether transfers
    // (there is no other function except the receive function).
    // Any call with non-empty calldata to this contract will execute
    // the fallback function (even if Ether is sent along with the call).
    fallback() external payable { x = 1; y = msg.value; }
// This function is called for plain Ether transfers, i.e.
    // for every call with empty calldata.
    receive() external payable { x = 2; y = msg.value; }
    uint x;
    uint y;
}
```

### Array Resizing
Access to length of arrays is now always read-only, even for storage arrays. It is no longer possible to resize storage arrays assigning a new value to their length. Use push(), push(value) or pop() instead, or assign a full array,

- Change `uint length = array.push(value)` to `array.push(value);`. The new length can be accessed via `array.length`.

- Change `array.length++` to `array.push()` to increase, and use `pop()` to decrease the length of a storage array.

```
pragma solidity ^0.6.0;
contract Test {
    string[] public names_A_to_F;
    function test() public {
        // invalid syntax
        uint length = names_A_to_F.push("Alice"); // invalid
        names_A_to_F.length++; // invalid as length is now read only
        // correct syntax
        names_A_to_F.push(); // increase array length
        names_A_to_F.push("Alice"); // add item to array
        names_A_to_F.pop(); // reduce array length by removing item
        uint length = names_A_to_F.length; // access array length
    }
}


```

### State variable shadowing is now disallowed
State variable shadowing is considered as an error. A derived contract can only declare a state variable x, if there is no visible state variable with the same name in any of its bases.

An example is shown below:

```
pragma solidity ^0.6.0;
contract Test {
    address x;
}
contract TestOverride is Test {
// This declaration would throw an error because its shadowing a
// state variable already in its base inherited contract Test
    address x;
}


```
This is an explicit requirement that furthers helps improve reading and understanding a Solidity codebase.

Source: [Coinmonks blog](https://medium.com/coinmonks/whats-new-in-solidity-0-6-56fa76198ec7)
