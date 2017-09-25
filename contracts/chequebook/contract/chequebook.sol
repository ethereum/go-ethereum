import "mortal";

/// @title Chequebook for Ethereum micropayments
/// @author Daniel A. Nagy <daniel@ethdev.com>
contract chequebook is mortal {
    // Cumulative paid amount in wei to each beneficiary
    mapping (address => uint256) public sent;

    /// @notice Overdraft event
    event Overdraft(address deadbeat);

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
        uint256 diff = amount - sent[beneficiary];
        if (diff <= this.balance) {
	    // update the cumulative amount before sending
            sent[beneficiary] = amount;
            if (!beneficiary.send(diff)) {
                // Upon failure to execute send, revert everything
                throw;
            }
        } else {
            // Upon failure, punish owner for writing a bounced cheque.
            // owner.sendToDebtorsPrison();
            Overdraft(owner);
            // Compensate beneficiary.
            suicide(beneficiary);
        }
    }
}
