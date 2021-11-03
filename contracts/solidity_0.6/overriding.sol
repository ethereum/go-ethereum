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
