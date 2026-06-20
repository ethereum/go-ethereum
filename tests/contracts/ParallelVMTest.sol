// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title ParallelVMTest
/// @notice Contract for testing declared read/write-set based parallel execution.
///
/// Conflict classes:
/// - isolatedJob(lane, ...) touches only lane-local storage
/// - contendedJob(...) always touches the same global slots
/// - mixedJob(lane, ...) touches both lane-local and global storage
///
/// Storage layout:
/// - laneValue[lane]  => keccak256(abi.encode(lane, 0))
/// - laneDigest[lane] => keccak256(abi.encode(lane, 1))
/// - globalValue      => slot 2
/// - globalDigest     => slot 3
contract ParallelVMTest {
    // mapping(uint256 => uint256) at slot 0
    mapping(uint256 => uint256) public laneValue;

    // mapping(uint256 => bytes32) at slot 1
    mapping(uint256 => bytes32) public laneDigest;

    // fixed slots
    uint256 public globalValue;   // slot 2
    bytes32 public globalDigest;  // slot 3

    event IsolatedJob(
        uint256 indexed lane,
        uint256 oldValue,
        uint256 newValue,
        bytes32 oldDigest,
        bytes32 newDigest
    );

    event ContendedJob(
        uint256 oldValue,
        uint256 newValue,
        bytes32 oldDigest,
        bytes32 newDigest
    );

    event MixedJob(
        uint256 indexed lane,
        uint256 oldLaneValue,
        uint256 newLaneValue,
        uint256 oldGlobalValue,
        uint256 newGlobalValue
    );

    /// @notice Touches only storage derived from `lane`.
    /// Different lanes should be non-conflicting.
    function isolatedJob(
        uint256 lane,
        uint256 addend,
        uint256 rounds
    ) external {
        uint256 oldValue = laneValue[lane];
        bytes32 oldDigest = laneDigest[lane];

        bytes32 newDigest = _work(
            keccak256(abi.encodePacked("isolated", lane, oldValue, oldDigest, addend)),
            rounds
        );

        uint256 newValue = oldValue + addend;

        laneValue[lane] = newValue;
        laneDigest[lane] = newDigest;

        emit IsolatedJob(lane, oldValue, newValue, oldDigest, newDigest);
    }

    /// @notice Always touches the same global slots.
    /// All calls conflict.
    function contendedJob(
        uint256 addend,
        uint256 rounds
    ) external {
        uint256 oldValue = globalValue;
        bytes32 oldDigest = globalDigest;

        bytes32 newDigest = _work(
            keccak256(abi.encodePacked("contended", oldValue, oldDigest, addend)),
            rounds
        );

        uint256 newValue = oldValue + addend;

        globalValue = newValue;
        globalDigest = newDigest;

        emit ContendedJob(oldValue, newValue, oldDigest, newDigest);
    }

    /// @notice Touches lane-local storage and shared global storage.
    /// Different lanes still conflict because of the shared globals.
    function mixedJob(
        uint256 lane,
        uint256 addend,
        uint256 rounds
    ) external {
        uint256 oldLaneValue = laneValue[lane];
        bytes32 oldLaneDigest = laneDigest[lane];
        uint256 oldGlobalValue = globalValue;
        bytes32 oldGlobalDigest = globalDigest;

        bytes32 laneNewDigest = _work(
            keccak256(
                abi.encodePacked(
                    "mixed-lane",
                    lane,
                    oldLaneValue,
                    oldLaneDigest,
                    addend
                )
            ),
            rounds
        );

        bytes32 globalNewDigest = _work(
            keccak256(
                abi.encodePacked(
                    "mixed-global",
                    oldGlobalValue,
                    oldGlobalDigest,
                    addend
                )
            ),
            rounds
        );

        laneValue[lane] = oldLaneValue + addend;
        laneDigest[lane] = laneNewDigest;

        globalValue = oldGlobalValue + addend;
        globalDigest = globalNewDigest;

        emit MixedJob(
            lane,
            oldLaneValue,
            oldLaneValue + addend,
            oldGlobalValue,
            oldGlobalValue + addend
        );
    }

    function slotForLaneValue(uint256 lane) external pure returns (bytes32) {
        return keccak256(abi.encode(lane, uint256(0)));
    }

    function slotForLaneDigest(uint256 lane) external pure returns (bytes32) {
        return keccak256(abi.encode(lane, uint256(1)));
    }

    function slotForGlobalValue() external pure returns (bytes32) {
        return bytes32(uint256(2));
    }

    function slotForGlobalDigest() external pure returns (bytes32) {
        return bytes32(uint256(3));
    }

    function readLane(uint256 lane) external view returns (uint256 value, bytes32 digest) {
        return (laneValue[lane], laneDigest[lane]);
    }

    function _work(bytes32 seed, uint256 rounds) internal pure returns (bytes32 x) {
        x = seed;
        unchecked {
            for (uint256 i = 0; i < rounds; i++) {
                x = keccak256(abi.encodePacked(x, i));
            }
        }
    }
}