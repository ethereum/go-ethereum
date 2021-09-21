pragma solidity ^0.4.24;

library SafeMath {

	/**
	 * @dev Multiplies two numbers, reverts on overflow.
	 */
	function mul(uint256 a, uint256 b) internal pure returns (uint256) {
		// Gas optimization: this is cheaper than requiring 'a' not being zero, but the
		// benefit is lost if 'b' is also tested.
		// See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
		if (a == 0) {
			return 0;
		}

		uint256 c = a * b;
		require(c / a == b);

		return c;
	}

	/**
	 * @dev Integer division of two numbers truncating the quotient, reverts on division by zero.
	 */
	function div(uint256 a, uint256 b) internal pure returns (uint256) {
		require(b > 0); // Solidity only automatically asserts when dividing by 0
		uint256 c = a / b;
		// assert(a == b * c + a % b); // There is no case in which this doesn't hold

		return c;
	}

    /**
     * @dev Subtracts two numbers, reverts on overflow (i.e. if subtrahend is greater than minuend).
     */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a);
        uint256 c = a - b;

        return c;
    }

    /**
     * @dev Adds two numbers, reverts on overflow.
     */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }

    /**
     * @dev Divides two numbers and returns the remainder (unsigned integer modulo),
     * reverts when dividing by zero.
     */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0);
        return a % b;
    }
}

contract AbstractTokenTRC21 {
    function issuer() public view returns (address);
}

contract TRC21Issuer {
    using SafeMath for uint256;
    uint256 _minCap;
    address[] _tokens;
    mapping(address => uint256) tokensState;

	event Apply(address indexed issuer, address indexed token, uint256 value);
	event Charge(address indexed supporter, address indexed token, uint256 value);

    constructor (uint256 value) public {
        _minCap = value;
    }

    function minCap() public view returns(uint256) {
        return _minCap;
    }

    function tokens() public view returns(address[]) {
        return _tokens;
    }

    function getTokenCapacity(address token) public view returns(uint256) {
        return tokensState[token];
    }

    modifier onlyValidCapacity(address token) {
        require(token != address(0));
        require(msg.value >= _minCap);
        _;
    }

    function apply(address token) public payable onlyValidCapacity(token) {
        AbstractTokenTRC21 t = AbstractTokenTRC21(token);
        require(t.issuer() == msg.sender);
        _tokens.push(token);
        tokensState[token] = tokensState[token].add(msg.value);
        emit Apply(msg.sender, token, msg.value);
    }

    function charge(address token) public payable {
        tokensState[token] = tokensState[token].add(msg.value);
        emit Charge(msg.sender, token, msg.value);
    }

}
