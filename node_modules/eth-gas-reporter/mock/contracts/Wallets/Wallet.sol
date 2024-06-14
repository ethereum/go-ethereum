pragma solidity ^0.5.0;

contract Wallet {

    event Deposit(address indexed _sender, uint _value);

    function transferPayment(uint payment, address payable recipient) public {
        address(recipient).transfer(payment);
    }

    function sendPayment(uint payment, address payable recipient) public {
        if (!address(recipient).send(payment))
            revert();
    }

    function getBalance() public view returns(uint){
        return address(this).balance;
    }

    function() external payable
    {
        if (msg.value > 0)
            emit Deposit(msg.sender, msg.value);
    }
}
