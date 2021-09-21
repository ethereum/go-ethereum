pragma solidity ^0.6.0;
abstract contract Feline {
    function utterance() public virtual returns (bytes32);
}
contract Cat is Feline {
    function utterance() public override returns (bytes32) {
        return "miaow";
     }
}
