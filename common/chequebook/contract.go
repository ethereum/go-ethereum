package chequebook

import ()

const (
	ContractCode = `606060405260008054600160a060020a03191633179055610201806100246000396000f3606060405260e060020a600035046341c0e1b58114610031578063d75d691d14610059578063fbf788d61461007e575b005b61002f60005433600160a060020a03908116911614156101ff57600054600160a060020a0316ff5b600160a060020a03600435166000908152600160205260409020546060908152602090f35b61002f600435602435604435606435608435600160a060020a03851660009081526001602052604081205485116100b8575b505050505050565b30600160a060020a039081166c0100000000000000000000000090810260609081529188160260745260888690526048812080825260ff8616608090815260a086905260c0859052909260019260e0926020928290866161da5a03f11561000257505060405151600054600160a060020a03908116911614610139576100b0565b85600160a060020a031660006001600050600089600160a060020a03168152602001908152602001600020600050548703604051809050600060405180830381858888f19350505050156101b357846001600050600088600160a060020a03168152602001908152602001600020600050819055506100b0565b60005460408051600160a060020a03929092168252517f2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f9789181900360200190a185600160a060020a0316ff5b56`

	ContractDeployedCode = `0x606060405260e060020a600035046341c0e1b58114610031578063d75d691d14610059578063fbf788d61461007e575b005b61002f60005433600160a060020a03908116911614156101ff57600054600160a060020a0316ff5b600160a060020a03600435166000908152600160205260409020546060908152602090f35b61002f600435602435604435606435608435600160a060020a03851660009081526001602052604081205485116100b8575b505050505050565b30600160a060020a039081166c0100000000000000000000000090810260609081529188160260745260888690526048812080825260ff8616608090815260a086905260c0859052909260019260e0926020928290866161da5a03f11561000257505060405151600054600160a060020a03908116911614610139576100b0565b85600160a060020a031660006001600050600089600160a060020a03168152602001908152602001600020600050548703604051809050600060405180830381858888f19350505050156101b357846001600050600088600160a060020a03168152602001908152602001600020600050819055506100b0565b60005460408051600160a060020a03929092168252517f2250e2993c15843b32621c89447cc589ee7a9f049c026986e545d3c2c0c6f9789181900360200190a185600160a060020a0316ff5b56`

	ContractAbi = `[{"constant":false,"inputs":[],"name":"kill","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"beneficiary","type":"address"}],"name":"getSent","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"beneficiary","type":"address"},{"name":"amount","type":"uint256"},{"name":"sig_v","type":"uint8"},{"name":"sig_r","type":"bytes32"},{"name":"sig_s","type":"bytes32"}],"name":"cash","outputs":[],"type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"deadbeat","type":"address"}],"name":"Overdraft","type":"event"}]`

	ContractSource = `
import "mortal";

/// @title Chequebook for Ethereum micropayments
/// @author Daniel A. Nagy <daniel@ethdev.com>
contract chequebook is mortal {
    // Cumulative paid amount in wei to each beneficiary
    mapping (address => uint256) sent;

    /// @notice Overdraft event
    event Overdraft(address deadbeat);

    /// @notice Accessor for sent map
    ///
    /// @param beneficiary beneficiary address
    /// @return cumulative amount in wei sent to beneficiary
    function getSent(address beneficiary) returns (uint256) {
      return sent[beneficiary];
    }

    /// @notice Cash cheque
    ///
    /// @param beneficiary beneficiary address
    /// @param amount cumulative amount in wei
    /// @param sig_v signature parameter v
    /// @param sig_r signature parameter r
    /// @param sig_s signature parameter s
    /// The digital signature is calculated on the concatenated triplet of contract address, beneficiary address and cumulative amount
    function cash(address beneficiary, uint256 amount,
        uint8 sig_v, bytes32 sig_r, bytes32 sig_s) {
        // Check if the cheque is old.
        // Only cheques that are more recent than the last cashed one are considered.
        if(amount <= sent[beneficiary]) return;
        // Check the digital signature of the cheque.
        bytes32 hash = sha3(address(this), beneficiary, amount);
        if(owner != ecrecover(hash, sig_v, sig_r, sig_s)) return;
        // Attempt sending the difference between the cumulative amount on the cheque
        // and the cumulative amount on the last cashed cheque to beneficiary.
        if (beneficiary.send(amount - sent[beneficiary])) {
            // Upon success, update the cumulative amount.
            sent[beneficiary] = amount;
        } else {
            // Upon failure, punish owner for writing a bounced cheque.
            // owner.sendToDebtorsPrison();
            Overdraft(owner);
            // Compensate beneficiary.
            suicide(beneficiary);
        }
    }
}
`
)
