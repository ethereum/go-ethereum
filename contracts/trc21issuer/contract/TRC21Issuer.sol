pragma solidity ^0.4.24;
import "./libs/SafeMath.sol";

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

    function charge(address token) public payable onlyValidCapacity(token) {
        tokensState[token] = tokensState[token].add(msg.value);
        emit Charge(msg.sender, token, msg.value);
    }

}
