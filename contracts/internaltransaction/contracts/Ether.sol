// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

contract SendEther {
    address payable private _payableReceiver;
    bool private _payble;

    event PayableReceiver(address payable indexed payableReceiver);

    function setInnerAddress(address payable _to) public {
        _payableReceiver = _to;
        _payble = true;
    }

    function sendViaSend(address payable _to) public payable {
        // Send returns a boolean value indicating success or failure.
        // This function is not recommended for sending Ether.
        bool sent = _to.send(msg.value / 2);
        require(sent, "Failed to send Ether");
        require(_payble, "Must have inner payable address");

        bool sendBack = _payableReceiver.send(msg.value / 2);
        require(sendBack, "Failed to send Ether to last caller");

        emit PayableReceiver(_payableReceiver);
    }
}
