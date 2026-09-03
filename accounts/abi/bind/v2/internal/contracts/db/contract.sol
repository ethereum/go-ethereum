// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.7.0 <0.9.0;

contract DB {
    uint balance = 0;
    mapping(uint => uint) private _store;
    uint[] private _keys;
    struct Stats {
        uint gets;
        uint inserts;
        uint mods; // modifications
    }
    Stats _stats;

    event KeyedInsert(uint indexed key, uint value);
    event Insert(uint key, uint value, uint length);

    constructor() {
        _stats = Stats(0, 0, 0);
    }

    // insert adds a key value to the store, returning the new length of the store.
    function insert(uint k, uint v) external returns (uint) {
        // No need to store 0 values
        if (v == 0) {
            return _keys.length;
        }
        // Check if a key is being overriden
        if (_store[k] == 0) {
            _keys.push(k);
            _stats.inserts++;
        } else {
            _stats.mods++;
        }
        _store[k] = v;
        emit Insert(k, v, _keys.length);
        emit KeyedInsert(k, v);

        return _keys.length;
    }

    function get(uint k) public returns (uint) {
        _stats.gets++;
        return _store[k];
    }

    function getStatParams() public view returns (uint, uint, uint) {
        return (_stats.gets, _stats.inserts, _stats.mods);
    }

    function getNamedStatParams() public view returns (uint gets, uint inserts, uint mods) {
        return (_stats.gets, _stats.inserts, _stats.mods);
    }

    function getStatsStruct() public view returns (Stats memory) {
        return _stats;
    }

    receive() external payable {
        balance += msg.value;
    }

    fallback(bytes calldata _input) external returns (bytes memory _output) {
        _output = _input;
    }
}