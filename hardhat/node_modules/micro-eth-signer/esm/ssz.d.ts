import * as P from 'micro-packed';
export declare const ForkSlots: {
    readonly Phase0: 0;
    readonly Altair: 2375680;
    readonly Bellatrix: 4700013;
    readonly Capella: 6209536;
    readonly Deneb: 8626176;
};
export type SSZCoder<T> = P.CoderType<T> & {
    default: T;
    info: {
        type: string;
    };
    composite: boolean;
    chunkCount: number;
    chunks: (value: T) => Uint8Array[];
    merkleRoot: (value: T) => Uint8Array;
    _isStableCompat: (other: SSZCoder<any>) => boolean;
};
export declare const uint8: SSZCoder<number | bigint>;
export declare const uint16: SSZCoder<number | bigint>;
export declare const uint32: SSZCoder<number | bigint>;
export declare const uint64: SSZCoder<number | bigint>;
export declare const uint128: SSZCoder<number | bigint>;
export declare const uint256: SSZCoder<number | bigint>;
export declare const boolean: SSZCoder<boolean>;
type VectorType<T> = SSZCoder<T[]> & {
    info: {
        type: 'vector';
        N: number;
        inner: SSZCoder<T>;
    };
};
/**
 * Vector: fixed size ('len') array of elements 'inner'
 */
export declare const vector: <T>(len: number, inner: SSZCoder<T>) => VectorType<T>;
type ListType<T> = SSZCoder<T[]> & {
    info: {
        type: 'list';
        N: number;
        inner: SSZCoder<T>;
    };
};
/**
 * List: dynamic array of 'inner' elements with length limit maxLen
 */
export declare const list: <T>(maxLen: number, inner: SSZCoder<T>) => ListType<T>;
type ContainerCoder<T extends Record<string, SSZCoder<any>>> = SSZCoder<{
    [K in keyof T]: P.UnwrapCoder<T[K]>;
}> & {
    info: {
        type: 'container';
        fields: T;
    };
};
/**
 * Container: Encodes object with multiple fields. P.struct for SSZ.
 */
export declare const container: <T extends Record<string, SSZCoder<any>>>(fields: T) => ContainerCoder<T>;
type BitVectorType = SSZCoder<boolean[]> & {
    info: {
        type: 'bitVector';
        N: number;
    };
};
/**
 * BitVector: array of booleans with fixed size
 */
export declare const bitvector: (len: number) => BitVectorType;
type BitListType = SSZCoder<boolean[]> & {
    info: {
        type: 'bitList';
        N: number;
    };
};
/**
 * BitList: array of booleans with dynamic size (but maxLen limit)
 */
export declare const bitlist: (maxLen: number) => BitListType;
/**
 * Union type (None is null)
 * */
export declare const union: (...types: (SSZCoder<any> | null)[]) => SSZCoder<{
    selector: number;
    value: any;
}>;
type ByteListType = SSZCoder<Uint8Array> & {
    info: {
        type: 'list';
        N: number;
        inner: typeof byte;
    };
};
/**
 * ByteList: same as List(len, SSZ.byte), but returns Uint8Array
 */
export declare const bytelist: (maxLen: number) => ByteListType;
type ByteVectorType = SSZCoder<Uint8Array> & {
    info: {
        type: 'vector';
        N: number;
        inner: typeof byte;
    };
};
/**
 * ByteVector: same as Vector(len, SSZ.byte), but returns Uint8Array
 */
export declare const bytevector: (len: number) => ByteVectorType;
type StableContainerCoder<T extends Record<string, SSZCoder<any>>> = SSZCoder<{
    [K in keyof T]?: P.UnwrapCoder<T[K]>;
}> & {
    info: {
        type: 'stableContainer';
        N: number;
        fields: T;
    };
};
/**
 * Same as container, but all values are optional using bitvector as prefix which indicates active fields
 */
export declare const stableContainer: <T extends Record<string, SSZCoder<any>>>(N: number, fields: T) => StableContainerCoder<T>;
type ProfileCoder<T extends Record<string, SSZCoder<any>>, OptK extends keyof T & string, ReqK extends keyof T & string> = SSZCoder<{
    [K in ReqK]: P.UnwrapCoder<T[K]>;
} & {
    [K in OptK]?: P.UnwrapCoder<T[K]>;
}> & {
    info: {
        type: 'profile';
        container: StableContainerCoder<T>;
    };
};
/**
 * Profile - fixed subset of stableContainer.
 * - fields and order of fields is exactly same as in underlying container
 * - some fields may be excluded or required in profile (all fields in stable container are always optional)
 * - adding new fields to underlying container won't change profile's constructed on top of it,
 *   because it is required to provide all list of optional fields.
 * - type of field can be changed inside profile (but we should be very explicit about this) to same shape type.
 *
 * @example
 * // class Shape(StableContainer[4]):
 * //     side: Optional[uint16]
 * //     color: Optional[uint8]
 * //     radius: Optional[uint16]
 *
 * // class Square(Profile[Shape]):
 * //     side: uint16
 * //     color: uint8
 *
 * // class Circle(Profile[Shape]):
 * //     color: uint8
 * //     radius: Optional[uint16]
 * // ->
 * const Shape = SSZ.stableContainer(4, {
 *   side: SSZ.uint16,
 *   color: SSZ.uint8,
 *   radius: SSZ.uint16,
 * });
 * const Square = profile(Shape, [], ['side', 'color']);
 * const Circle = profile(Shape, ['radius'], ['color']);
 * const Circle2 = profile(Shape, ['radius'], ['color'], { color: SSZ.byte });
 */
export declare const profile: <T extends Record<string, SSZCoder<any>>, OptK extends keyof T & string, ReqK extends keyof T & string>(c: StableContainerCoder<T>, optFields: OptK[], requiredFields?: ReqK[], replaceType?: Record<string, any>) => ProfileCoder<T, OptK, ReqK>;
export declare const byte: SSZCoder<number | bigint>;
export declare const bit: SSZCoder<boolean>;
export declare const bool: SSZCoder<boolean>;
export declare const bytes: (len: number) => ByteVectorType;
export declare const ETH2_TYPES: {
    Slot: SSZCoder<number | bigint>;
    Epoch: SSZCoder<number | bigint>;
    CommitteeIndex: SSZCoder<number | bigint>;
    ValidatorIndex: SSZCoder<number | bigint>;
    WithdrawalIndex: SSZCoder<number | bigint>;
    Gwei: SSZCoder<number | bigint>;
    Root: ByteVectorType;
    Hash32: ByteVectorType;
    Bytes32: ByteVectorType;
    Version: ByteVectorType;
    DomainType: ByteVectorType;
    ForkDigest: ByteVectorType;
    Domain: ByteVectorType;
    BLSPubkey: ByteVectorType;
    BLSSignature: ByteVectorType;
    Ether: SSZCoder<number | bigint>;
    ParticipationFlags: SSZCoder<number | bigint>;
    ExecutionAddress: ByteVectorType;
    PayloadId: ByteVectorType;
    KZGCommitment: ByteVectorType;
    KZGProof: ByteVectorType;
    Checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    AttestationData: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        index: SSZCoder<number | bigint>;
        beacon_block_root: ByteVectorType;
        source: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        target: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
    }>;
    Attestation: ContainerCoder<{
        aggregation_bits: BitListType;
        data: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            index: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            source: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            target: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
        }>;
        signature: ByteVectorType;
    }>;
    AggregateAndProof: ContainerCoder<{
        aggregator_index: SSZCoder<number | bigint>;
        aggregate: ContainerCoder<{
            aggregation_bits: BitListType;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }>;
        selection_proof: ByteVectorType;
    }>;
    IndexedAttestation: ContainerCoder<{
        attesting_indices: ListType<number | bigint>;
        data: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            index: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            source: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            target: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
        }>;
        signature: ByteVectorType;
    }>;
    AttesterSlashing: ContainerCoder<{
        attestation_1: ContainerCoder<{
            attesting_indices: ListType<number | bigint>;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }>;
        attestation_2: ContainerCoder<{
            attesting_indices: ListType<number | bigint>;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }>;
    }>;
    BLSToExecutionChange: ContainerCoder<{
        validator_index: SSZCoder<number | bigint>;
        from_bls_pubkey: ByteVectorType;
        to_execution_address: ByteVectorType;
    }>;
    ExecutionPayload: ContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions: ListType<Uint8Array<ArrayBufferLike>>;
        withdrawals: ListType<{
            index: number | bigint;
            validator_index: number | bigint;
            address: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
        }>;
        blob_gas_used: SSZCoder<number | bigint>;
        excess_blob_gas: SSZCoder<number | bigint>;
    }>;
    SyncAggregate: ContainerCoder<{
        sync_committee_bits: BitVectorType;
        sync_committee_signature: ByteVectorType;
    }>;
    VoluntaryExit: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        validator_index: SSZCoder<number | bigint>;
    }>;
    BeaconBlockHeader: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body_root: ByteVectorType;
    }>;
    SigningData: ContainerCoder<{
        object_root: ByteVectorType;
        domain: ByteVectorType;
    }>;
    SignedBeaconBlockHeader: ContainerCoder<{
        message: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            proposer_index: SSZCoder<number | bigint>;
            parent_root: ByteVectorType;
            state_root: ByteVectorType;
            body_root: ByteVectorType;
        }>;
        signature: ByteVectorType;
    }>;
    ProposerSlashing: ContainerCoder<{
        signed_header_1: ContainerCoder<{
            message: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            signature: ByteVectorType;
        }>;
        signed_header_2: ContainerCoder<{
            message: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            signature: ByteVectorType;
        }>;
    }>;
    DepositData: ContainerCoder<{
        pubkey: ByteVectorType;
        withdrawal_credentials: ByteVectorType;
        amount: SSZCoder<number | bigint>;
        signature: ByteVectorType;
    }>;
    Deposit: ContainerCoder<{
        proof: VectorType<Uint8Array<ArrayBufferLike>>;
        data: ContainerCoder<{
            pubkey: ByteVectorType;
            withdrawal_credentials: ByteVectorType;
            amount: SSZCoder<number | bigint>;
            signature: ByteVectorType;
        }>;
    }>;
    SignedVoluntaryExit: ContainerCoder<{
        message: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            validator_index: SSZCoder<number | bigint>;
        }>;
        signature: ByteVectorType;
    }>;
    Eth1Data: ContainerCoder<{
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
    }>;
    Withdrawal: ContainerCoder<{
        index: SSZCoder<number | bigint>;
        validator_index: SSZCoder<number | bigint>;
        address: ByteVectorType;
        amount: SSZCoder<number | bigint>;
    }>;
    BeaconBlockBody: ContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            attestation_2: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attestations: ListType<{
            aggregation_bits: boolean[];
            data: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        execution_payload: ContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions: ListType<Uint8Array<ArrayBufferLike>>;
            withdrawals: ListType<{
                index: number | bigint;
                validator_index: number | bigint;
                address: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
        }>;
        bls_to_execution_changes: ListType<{
            message: {
                validator_index: number | bigint;
                from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                to_execution_address: Uint8Array<ArrayBufferLike>;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        blob_kzg_commitments: ListType<Uint8Array<ArrayBufferLike>>;
    }>;
    BeaconBlock: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body: ContainerCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                attestation_2: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attestations: ListType<{
                aggregation_bits: boolean[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            sync_aggregate: ContainerCoder<{
                sync_committee_bits: BitVectorType;
                sync_committee_signature: ByteVectorType;
            }>;
            execution_payload: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions: ListType<Uint8Array<ArrayBufferLike>>;
                withdrawals: ListType<{
                    index: number | bigint;
                    validator_index: number | bigint;
                    address: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                }>;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            bls_to_execution_changes: ListType<{
                message: {
                    validator_index: number | bigint;
                    from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                    to_execution_address: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            blob_kzg_commitments: ListType<Uint8Array<ArrayBufferLike>>;
        }>;
    }>;
    SyncCommittee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    Fork: ContainerCoder<{
        previous_version: ByteVectorType;
        current_version: ByteVectorType;
        epoch: SSZCoder<number | bigint>;
    }>;
    Validator: ContainerCoder<{
        pubkey: ByteVectorType;
        withdrawal_credentials: ByteVectorType;
        effective_balance: SSZCoder<number | bigint>;
        slashed: SSZCoder<boolean>;
        activation_eligibility_epoch: SSZCoder<number | bigint>;
        activation_epoch: SSZCoder<number | bigint>;
        exit_epoch: SSZCoder<number | bigint>;
        withdrawable_epoch: SSZCoder<number | bigint>;
    }>;
    ExecutionPayloadHeader: ContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions_root: ByteVectorType;
        withdrawals_root: ByteVectorType;
        blob_gas_used: SSZCoder<number | bigint>;
        excess_blob_gas: SSZCoder<number | bigint>;
    }>;
    HistoricalSummary: ContainerCoder<{
        block_summary_root: ByteVectorType;
        state_summary_root: ByteVectorType;
    }>;
    BeaconState: ContainerCoder<{
        genesis_time: SSZCoder<number | bigint>;
        genesis_validators_root: ByteVectorType;
        slot: SSZCoder<number | bigint>;
        fork: ContainerCoder<{
            previous_version: ByteVectorType;
            current_version: ByteVectorType;
            epoch: SSZCoder<number | bigint>;
        }>;
        latest_block_header: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            proposer_index: SSZCoder<number | bigint>;
            parent_root: ByteVectorType;
            state_root: ByteVectorType;
            body_root: ByteVectorType;
        }>;
        block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
        state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
        historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        eth1_data_votes: ListType<{
            deposit_root: Uint8Array<ArrayBufferLike>;
            deposit_count: number | bigint;
            block_hash: Uint8Array<ArrayBufferLike>;
        }>;
        eth1_deposit_index: SSZCoder<number | bigint>;
        validators: ListType<{
            pubkey: Uint8Array<ArrayBufferLike>;
            withdrawal_credentials: Uint8Array<ArrayBufferLike>;
            effective_balance: number | bigint;
            slashed: boolean;
            activation_eligibility_epoch: number | bigint;
            activation_epoch: number | bigint;
            exit_epoch: number | bigint;
            withdrawable_epoch: number | bigint;
        }>;
        balances: ListType<number | bigint>;
        randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
        slashings: VectorType<number | bigint>;
        previous_epoch_participation: ListType<number | bigint>;
        current_epoch_participation: ListType<number | bigint>;
        justification_bits: BitVectorType;
        previous_justified_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        current_justified_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        finalized_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        inactivity_scores: ListType<number | bigint>;
        current_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        next_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        latest_execution_payload_header: ContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
            withdrawals_root: ByteVectorType;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
        }>;
        next_withdrawal_index: SSZCoder<number | bigint>;
        next_withdrawal_validator_index: SSZCoder<number | bigint>;
        historical_summaries: ListType<{
            block_summary_root: Uint8Array<ArrayBufferLike>;
            state_summary_root: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
    BlobIdentifier: ContainerCoder<{
        block_root: ByteVectorType;
        index: SSZCoder<number | bigint>;
    }>;
    BlobSidecar: ContainerCoder<{
        index: SSZCoder<number | bigint>;
        blob: ByteVectorType;
        kzg_commitment: ByteVectorType;
        kzg_proof: ByteVectorType;
        signed_block_header: ContainerCoder<{
            message: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            signature: ByteVectorType;
        }>;
        kzg_commitment_inclusion_proof: VectorType<Uint8Array<ArrayBufferLike>>;
    }>;
    ContributionAndProof: ContainerCoder<{
        aggregator_index: SSZCoder<number | bigint>;
        contribution: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            subcommittee_index: SSZCoder<number | bigint>;
            aggregation_bits: BitVectorType;
            signature: ByteVectorType;
        }>;
        selection_proof: ByteVectorType;
    }>;
    DepositMessage: ContainerCoder<{
        pubkey: ByteVectorType;
        withdrawal_credentials: ByteVectorType;
        amount: SSZCoder<number | bigint>;
    }>;
    Eth1Block: ContainerCoder<{
        timestamp: SSZCoder<number | bigint>;
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
    }>;
    ForkData: ContainerCoder<{
        current_version: ByteVectorType;
        genesis_validators_root: ByteVectorType;
    }>;
    HistoricalBatch: ContainerCoder<{
        block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
        state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    }>;
    PendingAttestation: ContainerCoder<{
        aggregation_bits: BitListType;
        data: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            index: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            source: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            target: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
        }>;
        inclusion_delay: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
    }>;
    PowBlock: ContainerCoder<{
        block_hash: ByteVectorType;
        parent_hash: ByteVectorType;
        total_difficulty: SSZCoder<number | bigint>;
    }>;
    Transaction: ByteListType;
    SignedAggregateAndProof: ContainerCoder<{
        message: ContainerCoder<{
            aggregator_index: SSZCoder<number | bigint>;
            aggregate: ContainerCoder<{
                aggregation_bits: BitListType;
                data: ContainerCoder<{
                    slot: SSZCoder<number | bigint>;
                    index: SSZCoder<number | bigint>;
                    beacon_block_root: ByteVectorType;
                    source: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                    target: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                }>;
                signature: ByteVectorType;
            }>;
            selection_proof: ByteVectorType;
        }>;
        signature: ByteVectorType;
    }>;
    SignedBLSToExecutionChange: ContainerCoder<{
        message: ContainerCoder<{
            validator_index: SSZCoder<number | bigint>;
            from_bls_pubkey: ByteVectorType;
            to_execution_address: ByteVectorType;
        }>;
        signature: ByteVectorType;
    }>;
    SignedBeaconBlock: ContainerCoder<{
        message: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            proposer_index: SSZCoder<number | bigint>;
            parent_root: ByteVectorType;
            state_root: ByteVectorType;
            body: ContainerCoder<{
                randao_reveal: ByteVectorType;
                eth1_data: ContainerCoder<{
                    deposit_root: ByteVectorType;
                    deposit_count: SSZCoder<number | bigint>;
                    block_hash: ByteVectorType;
                }>;
                graffiti: ByteVectorType;
                proposer_slashings: ListType<{
                    signed_header_1: {
                        message: {
                            slot: number | bigint;
                            proposer_index: number | bigint;
                            parent_root: Uint8Array<ArrayBufferLike>;
                            state_root: Uint8Array<ArrayBufferLike>;
                            body_root: Uint8Array<ArrayBufferLike>;
                        };
                        signature: Uint8Array<ArrayBufferLike>;
                    };
                    signed_header_2: {
                        message: {
                            slot: number | bigint;
                            proposer_index: number | bigint;
                            parent_root: Uint8Array<ArrayBufferLike>;
                            state_root: Uint8Array<ArrayBufferLike>;
                            body_root: Uint8Array<ArrayBufferLike>;
                        };
                        signature: Uint8Array<ArrayBufferLike>;
                    };
                }>;
                attester_slashings: ListType<{
                    attestation_1: {
                        attesting_indices: (number | bigint)[];
                        data: {
                            slot: number | bigint;
                            index: number | bigint;
                            beacon_block_root: Uint8Array<ArrayBufferLike>;
                            source: {
                                epoch: number | bigint;
                                root: Uint8Array<ArrayBufferLike>;
                            };
                            target: {
                                epoch: number | bigint;
                                root: Uint8Array<ArrayBufferLike>;
                            };
                        };
                        signature: Uint8Array<ArrayBufferLike>;
                    };
                    attestation_2: {
                        attesting_indices: (number | bigint)[];
                        data: {
                            slot: number | bigint;
                            index: number | bigint;
                            beacon_block_root: Uint8Array<ArrayBufferLike>;
                            source: {
                                epoch: number | bigint;
                                root: Uint8Array<ArrayBufferLike>;
                            };
                            target: {
                                epoch: number | bigint;
                                root: Uint8Array<ArrayBufferLike>;
                            };
                        };
                        signature: Uint8Array<ArrayBufferLike>;
                    };
                }>;
                attestations: ListType<{
                    aggregation_bits: boolean[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                }>;
                deposits: ListType<{
                    proof: Uint8Array<ArrayBufferLike>[];
                    data: {
                        pubkey: Uint8Array<ArrayBufferLike>;
                        withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                        amount: number | bigint;
                        signature: Uint8Array<ArrayBufferLike>;
                    };
                }>;
                voluntary_exits: ListType<{
                    message: {
                        epoch: number | bigint;
                        validator_index: number | bigint;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                }>;
                sync_aggregate: ContainerCoder<{
                    sync_committee_bits: BitVectorType;
                    sync_committee_signature: ByteVectorType;
                }>;
                execution_payload: ContainerCoder<{
                    parent_hash: ByteVectorType;
                    fee_recipient: ByteVectorType;
                    state_root: ByteVectorType;
                    receipts_root: ByteVectorType;
                    logs_bloom: ByteVectorType;
                    prev_randao: ByteVectorType;
                    block_number: SSZCoder<number | bigint>;
                    gas_limit: SSZCoder<number | bigint>;
                    gas_used: SSZCoder<number | bigint>;
                    timestamp: SSZCoder<number | bigint>;
                    extra_data: ByteListType;
                    base_fee_per_gas: SSZCoder<number | bigint>;
                    block_hash: ByteVectorType;
                    transactions: ListType<Uint8Array<ArrayBufferLike>>;
                    withdrawals: ListType<{
                        index: number | bigint;
                        validator_index: number | bigint;
                        address: Uint8Array<ArrayBufferLike>;
                        amount: number | bigint;
                    }>;
                    blob_gas_used: SSZCoder<number | bigint>;
                    excess_blob_gas: SSZCoder<number | bigint>;
                }>;
                bls_to_execution_changes: ListType<{
                    message: {
                        validator_index: number | bigint;
                        from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                        to_execution_address: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                }>;
                blob_kzg_commitments: ListType<Uint8Array<ArrayBufferLike>>;
            }>;
        }>;
        signature: ByteVectorType;
    }>;
    SignedContributionAndProof: ContainerCoder<{
        message: ContainerCoder<{
            aggregator_index: SSZCoder<number | bigint>;
            contribution: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                subcommittee_index: SSZCoder<number | bigint>;
                aggregation_bits: BitVectorType;
                signature: ByteVectorType;
            }>;
            selection_proof: ByteVectorType;
        }>;
        signature: ByteVectorType;
    }>;
    SyncAggregatorSelectionData: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        subcommittee_index: SSZCoder<number | bigint>;
    }>;
    SyncCommitteeContribution: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        beacon_block_root: ByteVectorType;
        subcommittee_index: SSZCoder<number | bigint>;
        aggregation_bits: BitVectorType;
        signature: ByteVectorType;
    }>;
    SyncCommitteeMessage: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        beacon_block_root: ByteVectorType;
        validator_index: SSZCoder<number | bigint>;
        signature: ByteVectorType;
    }>;
    LightClientHeader: ContainerCoder<{
        beacon: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            proposer_index: SSZCoder<number | bigint>;
            parent_root: ByteVectorType;
            state_root: ByteVectorType;
            body_root: ByteVectorType;
        }>;
        execution: ContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
            withdrawals_root: ByteVectorType;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
        }>;
        execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
    }>;
    LightClientBootstrap: ContainerCoder<{
        header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        current_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        current_sync_committee_branch: VectorType<Uint8Array<ArrayBufferLike>>;
    }>;
    LightClientUpdate: ContainerCoder<{
        attested_header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        next_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        next_sync_committee_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        finalized_header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        finality_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        signature_slot: SSZCoder<number | bigint>;
    }>;
    LightClientOptimisticUpdate: ContainerCoder<{
        attested_header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        signature_slot: SSZCoder<number | bigint>;
    }>;
    LightClientFinalityUpdate: ContainerCoder<{
        attested_header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        finalized_header: ContainerCoder<{
            beacon: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            execution: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
            }>;
            execution_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        }>;
        finality_branch: VectorType<Uint8Array<ArrayBufferLike>>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        signature_slot: SSZCoder<number | bigint>;
    }>;
    DepositRequest: ContainerCoder<{
        pubkey: ByteVectorType;
        withdrawal_credentials: ByteVectorType;
        amount: SSZCoder<number | bigint>;
        signature: ByteVectorType;
        index: SSZCoder<number | bigint>;
    }>;
    WithdrawalRequest: ContainerCoder<{
        source_address: ByteVectorType;
        validator_pubkey: ByteVectorType;
        amount: SSZCoder<number | bigint>;
    }>;
    ConsolidationRequest: ContainerCoder<{
        source_address: ByteVectorType;
        source_pubkey: ByteVectorType;
        target_pubkey: ByteVectorType;
    }>;
    PendingBalanceDeposit: ContainerCoder<{
        index: SSZCoder<number | bigint>;
        amount: SSZCoder<number | bigint>;
    }>;
    PendingPartialWithdrawal: ContainerCoder<{
        index: SSZCoder<number | bigint>;
        amount: SSZCoder<number | bigint>;
        withdrawable_epoch: SSZCoder<number | bigint>;
    }>;
    PendingConsolidation: ContainerCoder<{
        source_index: SSZCoder<number | bigint>;
        target_index: SSZCoder<number | bigint>;
    }>;
};
export declare const ETH2_CONSENSUS: {
    StableAttestation: StableContainerCoder<{
        aggregation_bits: BitListType;
        data: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            index: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            source: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            target: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
        }>;
        signature: ByteVectorType;
        committee_bits: BitVectorType;
    }>;
    StableIndexedAttestation: StableContainerCoder<{
        attesting_indices: ListType<number | bigint>;
        data: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            index: SSZCoder<number | bigint>;
            beacon_block_root: ByteVectorType;
            source: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            target: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
        }>;
        signature: ByteVectorType;
    }>;
    StableAttesterSlashing: ContainerCoder<{
        attestation_1: StableContainerCoder<{
            attesting_indices: ListType<number | bigint>;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }>;
        attestation_2: StableContainerCoder<{
            attesting_indices: ListType<number | bigint>;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }>;
    }>;
    StableExecutionPayload: StableContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions: ListType<Uint8Array<ArrayBufferLike>>;
        withdrawals: ListType<{
            index: number | bigint;
            validator_index: number | bigint;
            address: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
        }>;
        blob_gas_used: SSZCoder<number | bigint>;
        excess_blob_gas: SSZCoder<number | bigint>;
        deposit_requests: ListType<{
            pubkey: Uint8Array<ArrayBufferLike>;
            withdrawal_credentials: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
            signature: Uint8Array<ArrayBufferLike>;
            index: number | bigint;
        }>;
        withdrawal_requests: ListType<{
            source_address: Uint8Array<ArrayBufferLike>;
            validator_pubkey: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
        }>;
        consolidation_requests: ListType<{
            source_address: Uint8Array<ArrayBufferLike>;
            source_pubkey: Uint8Array<ArrayBufferLike>;
            target_pubkey: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
    StableExecutionRequests: StableContainerCoder<{
        deposits: ListType<{
            pubkey: Uint8Array<ArrayBufferLike>;
            withdrawal_credentials: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
            signature: Uint8Array<ArrayBufferLike>;
            index: number | bigint;
        }>;
        withdrawals: ListType<{
            source_address: Uint8Array<ArrayBufferLike>;
            validator_pubkey: Uint8Array<ArrayBufferLike>;
            amount: number | bigint;
        }>;
        consolidations: ListType<{
            source_address: Uint8Array<ArrayBufferLike>;
            source_pubkey: Uint8Array<ArrayBufferLike>;
            target_pubkey: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
    StableExecutionPayloadHeader: StableContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions_root: ByteVectorType;
        withdrawals_root: ByteVectorType;
        blob_gas_used: SSZCoder<number | bigint>;
        excess_blob_gas: SSZCoder<number | bigint>;
        deposit_requests_root: ByteVectorType;
        withdrawal_requests_root: ByteVectorType;
        consolidation_requests_root: ByteVectorType;
    }>;
    StableBeaconBlockBody: StableContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices?: (number | bigint)[] | undefined;
                data?: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                } | undefined;
                signature?: Uint8Array<ArrayBufferLike> | undefined;
            };
            attestation_2: {
                attesting_indices?: (number | bigint)[] | undefined;
                data?: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                } | undefined;
                signature?: Uint8Array<ArrayBufferLike> | undefined;
            };
        }>;
        attestations: ListType<{
            aggregation_bits?: boolean[] | undefined;
            data?: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            } | undefined;
            signature?: Uint8Array<ArrayBufferLike> | undefined;
            committee_bits?: boolean[] | undefined;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        execution_payload: StableContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions: ListType<Uint8Array<ArrayBufferLike>>;
            withdrawals: ListType<{
                index: number | bigint;
                validator_index: number | bigint;
                address: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
            deposit_requests: ListType<{
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
                index: number | bigint;
            }>;
            withdrawal_requests: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                validator_pubkey: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            consolidation_requests: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                source_pubkey: Uint8Array<ArrayBufferLike>;
                target_pubkey: Uint8Array<ArrayBufferLike>;
            }>;
        }>;
        bls_to_execution_changes: ListType<{
            message: {
                validator_index: number | bigint;
                from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                to_execution_address: Uint8Array<ArrayBufferLike>;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        blob_kzg_commitments: ListType<Uint8Array<ArrayBufferLike>>;
        execution_requests: StableContainerCoder<{
            deposits: ListType<{
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
                index: number | bigint;
            }>;
            withdrawals: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                validator_pubkey: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            consolidations: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                source_pubkey: Uint8Array<ArrayBufferLike>;
                target_pubkey: Uint8Array<ArrayBufferLike>;
            }>;
        }>;
    }>;
    StableBeaconState: StableContainerCoder<{
        genesis_time: SSZCoder<number | bigint>;
        genesis_validators_root: ByteVectorType;
        slot: SSZCoder<number | bigint>;
        fork: ContainerCoder<{
            previous_version: ByteVectorType;
            current_version: ByteVectorType;
            epoch: SSZCoder<number | bigint>;
        }>;
        latest_block_header: ContainerCoder<{
            slot: SSZCoder<number | bigint>;
            proposer_index: SSZCoder<number | bigint>;
            parent_root: ByteVectorType;
            state_root: ByteVectorType;
            body_root: ByteVectorType;
        }>;
        block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
        state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
        historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        eth1_data_votes: ListType<{
            deposit_root: Uint8Array<ArrayBufferLike>;
            deposit_count: number | bigint;
            block_hash: Uint8Array<ArrayBufferLike>;
        }>;
        eth1_deposit_index: SSZCoder<number | bigint>;
        validators: ListType<{
            pubkey: Uint8Array<ArrayBufferLike>;
            withdrawal_credentials: Uint8Array<ArrayBufferLike>;
            effective_balance: number | bigint;
            slashed: boolean;
            activation_eligibility_epoch: number | bigint;
            activation_epoch: number | bigint;
            exit_epoch: number | bigint;
            withdrawable_epoch: number | bigint;
        }>;
        balances: ListType<number | bigint>;
        randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
        slashings: VectorType<number | bigint>;
        previous_epoch_participation: ListType<number | bigint>;
        current_epoch_participation: ListType<number | bigint>;
        justification_bits: BitVectorType;
        previous_justified_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        current_justified_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        finalized_checkpoint: ContainerCoder<{
            epoch: SSZCoder<number | bigint>;
            root: ByteVectorType;
        }>;
        inactivity_scores: ListType<number | bigint>;
        current_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        next_sync_committee: ContainerCoder<{
            pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
            aggregate_pubkey: ByteVectorType;
        }>;
        latest_execution_payload_header: StableContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
            withdrawals_root: ByteVectorType;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
            deposit_requests_root: ByteVectorType;
            withdrawal_requests_root: ByteVectorType;
            consolidation_requests_root: ByteVectorType;
        }>;
        next_withdrawal_index: SSZCoder<number | bigint>;
        next_withdrawal_validator_index: SSZCoder<number | bigint>;
        historical_summaries: ListType<{
            block_summary_root: Uint8Array<ArrayBufferLike>;
            state_summary_root: Uint8Array<ArrayBufferLike>;
        }>;
        deposit_requests_start_index: SSZCoder<number | bigint>;
        deposit_balance_to_consume: SSZCoder<number | bigint>;
        exit_balance_to_consume: SSZCoder<number | bigint>;
        earliest_exit_epoch: SSZCoder<number | bigint>;
        consolidation_balance_to_consume: SSZCoder<number | bigint>;
        earliest_consolidation_epoch: SSZCoder<number | bigint>;
        pending_balance_deposits: ListType<{
            index: number | bigint;
            amount: number | bigint;
        }>;
        pending_partial_withdrawals: ListType<{
            index: number | bigint;
            amount: number | bigint;
            withdrawable_epoch: number | bigint;
        }>;
        pending_consolidations: ListType<{
            source_index: number | bigint;
            target_index: number | bigint;
        }>;
    }>;
};
export declare const ETH2_PROFILES: {
    electra: {
        Attestation: ProfileCoder<{
            aggregation_bits: BitListType;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
            committee_bits: BitVectorType;
        }, never, "data" | "signature" | "aggregation_bits" | "committee_bits">;
        AttesterSlashing: ContainerCoder<{
            attestation_1: ProfileCoder<{
                attesting_indices: ListType<number | bigint>;
                data: ContainerCoder<{
                    slot: SSZCoder<number | bigint>;
                    index: SSZCoder<number | bigint>;
                    beacon_block_root: ByteVectorType;
                    source: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                    target: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                }>;
                signature: ByteVectorType;
            }, never, "data" | "signature" | "attesting_indices">;
            attestation_2: ProfileCoder<{
                attesting_indices: ListType<number | bigint>;
                data: ContainerCoder<{
                    slot: SSZCoder<number | bigint>;
                    index: SSZCoder<number | bigint>;
                    beacon_block_root: ByteVectorType;
                    source: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                    target: ContainerCoder<{
                        epoch: SSZCoder<number | bigint>;
                        root: ByteVectorType;
                    }>;
                }>;
                signature: ByteVectorType;
            }, never, "data" | "signature" | "attesting_indices">;
        }>;
        IndexedAttestation: ProfileCoder<{
            attesting_indices: ListType<number | bigint>;
            data: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                index: SSZCoder<number | bigint>;
                beacon_block_root: ByteVectorType;
                source: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
                target: ContainerCoder<{
                    epoch: SSZCoder<number | bigint>;
                    root: ByteVectorType;
                }>;
            }>;
            signature: ByteVectorType;
        }, never, "data" | "signature" | "attesting_indices">;
        ExecutionRequests: ProfileCoder<{
            deposits: ListType<{
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
                index: number | bigint;
            }>;
            withdrawals: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                validator_pubkey: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            consolidations: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                source_pubkey: Uint8Array<ArrayBufferLike>;
                target_pubkey: Uint8Array<ArrayBufferLike>;
            }>;
        }, never, "withdrawals" | "deposits" | "consolidations">;
        ExecutionPayloadHeader: ProfileCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
            withdrawals_root: ByteVectorType;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
            deposit_requests_root: ByteVectorType;
            withdrawal_requests_root: ByteVectorType;
            consolidation_requests_root: ByteVectorType;
        }, never, "parent_hash" | "fee_recipient" | "state_root" | "receipts_root" | "logs_bloom" | "prev_randao" | "block_number" | "gas_limit" | "gas_used" | "timestamp" | "extra_data" | "base_fee_per_gas" | "block_hash" | "blob_gas_used" | "excess_blob_gas" | "transactions_root" | "withdrawals_root">;
        ExecutionPayload: ProfileCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions: ListType<Uint8Array<ArrayBufferLike>>;
            withdrawals: ListType<{
                index: number | bigint;
                validator_index: number | bigint;
                address: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            blob_gas_used: SSZCoder<number | bigint>;
            excess_blob_gas: SSZCoder<number | bigint>;
            deposit_requests: ListType<{
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
                index: number | bigint;
            }>;
            withdrawal_requests: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                validator_pubkey: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
            }>;
            consolidation_requests: ListType<{
                source_address: Uint8Array<ArrayBufferLike>;
                source_pubkey: Uint8Array<ArrayBufferLike>;
                target_pubkey: Uint8Array<ArrayBufferLike>;
            }>;
        }, never, "parent_hash" | "fee_recipient" | "state_root" | "receipts_root" | "logs_bloom" | "prev_randao" | "block_number" | "gas_limit" | "gas_used" | "timestamp" | "extra_data" | "base_fee_per_gas" | "block_hash" | "transactions" | "withdrawals" | "blob_gas_used" | "excess_blob_gas">;
        BeaconBlockBody: ProfileCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices?: (number | bigint)[] | undefined;
                    data?: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    } | undefined;
                    signature?: Uint8Array<ArrayBufferLike> | undefined;
                };
                attestation_2: {
                    attesting_indices?: (number | bigint)[] | undefined;
                    data?: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    } | undefined;
                    signature?: Uint8Array<ArrayBufferLike> | undefined;
                };
            }>;
            attestations: ListType<{
                aggregation_bits?: boolean[] | undefined;
                data?: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                } | undefined;
                signature?: Uint8Array<ArrayBufferLike> | undefined;
                committee_bits?: boolean[] | undefined;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            sync_aggregate: ContainerCoder<{
                sync_committee_bits: BitVectorType;
                sync_committee_signature: ByteVectorType;
            }>;
            execution_payload: StableContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions: ListType<Uint8Array<ArrayBufferLike>>;
                withdrawals: ListType<{
                    index: number | bigint;
                    validator_index: number | bigint;
                    address: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                }>;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
                deposit_requests: ListType<{
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                    index: number | bigint;
                }>;
                withdrawal_requests: ListType<{
                    source_address: Uint8Array<ArrayBufferLike>;
                    validator_pubkey: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                }>;
                consolidation_requests: ListType<{
                    source_address: Uint8Array<ArrayBufferLike>;
                    source_pubkey: Uint8Array<ArrayBufferLike>;
                    target_pubkey: Uint8Array<ArrayBufferLike>;
                }>;
            }>;
            bls_to_execution_changes: ListType<{
                message: {
                    validator_index: number | bigint;
                    from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                    to_execution_address: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            blob_kzg_commitments: ListType<Uint8Array<ArrayBufferLike>>;
            execution_requests: StableContainerCoder<{
                deposits: ListType<{
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                    index: number | bigint;
                }>;
                withdrawals: ListType<{
                    source_address: Uint8Array<ArrayBufferLike>;
                    validator_pubkey: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                }>;
                consolidations: ListType<{
                    source_address: Uint8Array<ArrayBufferLike>;
                    source_pubkey: Uint8Array<ArrayBufferLike>;
                    target_pubkey: Uint8Array<ArrayBufferLike>;
                }>;
            }>;
        }, never, "randao_reveal" | "eth1_data" | "graffiti" | "proposer_slashings" | "attester_slashings" | "attestations" | "deposits" | "voluntary_exits" | "sync_aggregate" | "execution_payload" | "bls_to_execution_changes" | "blob_kzg_commitments" | "execution_requests">;
        BeaconState: ProfileCoder<{
            genesis_time: SSZCoder<number | bigint>;
            genesis_validators_root: ByteVectorType;
            slot: SSZCoder<number | bigint>;
            fork: ContainerCoder<{
                previous_version: ByteVectorType;
                current_version: ByteVectorType;
                epoch: SSZCoder<number | bigint>;
            }>;
            latest_block_header: ContainerCoder<{
                slot: SSZCoder<number | bigint>;
                proposer_index: SSZCoder<number | bigint>;
                parent_root: ByteVectorType;
                state_root: ByteVectorType;
                body_root: ByteVectorType;
            }>;
            block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
            state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
            historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            eth1_data_votes: ListType<{
                deposit_root: Uint8Array<ArrayBufferLike>;
                deposit_count: number | bigint;
                block_hash: Uint8Array<ArrayBufferLike>;
            }>;
            eth1_deposit_index: SSZCoder<number | bigint>;
            validators: ListType<{
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                effective_balance: number | bigint;
                slashed: boolean;
                activation_eligibility_epoch: number | bigint;
                activation_epoch: number | bigint;
                exit_epoch: number | bigint;
                withdrawable_epoch: number | bigint;
            }>;
            balances: ListType<number | bigint>;
            randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
            slashings: VectorType<number | bigint>;
            previous_epoch_participation: ListType<number | bigint>;
            current_epoch_participation: ListType<number | bigint>;
            justification_bits: BitVectorType;
            previous_justified_checkpoint: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            current_justified_checkpoint: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            finalized_checkpoint: ContainerCoder<{
                epoch: SSZCoder<number | bigint>;
                root: ByteVectorType;
            }>;
            inactivity_scores: ListType<number | bigint>;
            current_sync_committee: ContainerCoder<{
                pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
                aggregate_pubkey: ByteVectorType;
            }>;
            next_sync_committee: ContainerCoder<{
                pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
                aggregate_pubkey: ByteVectorType;
            }>;
            latest_execution_payload_header: StableContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
                blob_gas_used: SSZCoder<number | bigint>;
                excess_blob_gas: SSZCoder<number | bigint>;
                deposit_requests_root: ByteVectorType;
                withdrawal_requests_root: ByteVectorType;
                consolidation_requests_root: ByteVectorType;
            }>;
            next_withdrawal_index: SSZCoder<number | bigint>;
            next_withdrawal_validator_index: SSZCoder<number | bigint>;
            historical_summaries: ListType<{
                block_summary_root: Uint8Array<ArrayBufferLike>;
                state_summary_root: Uint8Array<ArrayBufferLike>;
            }>;
            deposit_requests_start_index: SSZCoder<number | bigint>;
            deposit_balance_to_consume: SSZCoder<number | bigint>;
            exit_balance_to_consume: SSZCoder<number | bigint>;
            earliest_exit_epoch: SSZCoder<number | bigint>;
            consolidation_balance_to_consume: SSZCoder<number | bigint>;
            earliest_consolidation_epoch: SSZCoder<number | bigint>;
            pending_balance_deposits: ListType<{
                index: number | bigint;
                amount: number | bigint;
            }>;
            pending_partial_withdrawals: ListType<{
                index: number | bigint;
                amount: number | bigint;
                withdrawable_epoch: number | bigint;
            }>;
            pending_consolidations: ListType<{
                source_index: number | bigint;
                target_index: number | bigint;
            }>;
        }, never, "slot" | "eth1_data" | "genesis_time" | "genesis_validators_root" | "fork" | "latest_block_header" | "block_roots" | "state_roots" | "historical_roots" | "eth1_data_votes" | "eth1_deposit_index" | "validators" | "balances" | "randao_mixes" | "slashings" | "previous_epoch_participation" | "current_epoch_participation" | "justification_bits" | "previous_justified_checkpoint" | "current_justified_checkpoint" | "finalized_checkpoint" | "inactivity_scores" | "current_sync_committee" | "next_sync_committee" | "latest_execution_payload_header" | "next_withdrawal_index" | "next_withdrawal_validator_index" | "historical_summaries" | "deposit_requests_start_index" | "deposit_balance_to_consume" | "exit_balance_to_consume" | "earliest_exit_epoch" | "consolidation_balance_to_consume" | "earliest_consolidation_epoch" | "pending_balance_deposits" | "pending_partial_withdrawals" | "pending_consolidations">;
    };
};
/** Capella Types */
export declare const CapellaExecutionPayloadHeader: ContainerCoder<{
    parent_hash: ByteVectorType;
    fee_recipient: ByteVectorType;
    state_root: ByteVectorType;
    receipts_root: ByteVectorType;
    logs_bloom: ByteVectorType;
    prev_randao: ByteVectorType;
    block_number: SSZCoder<number | bigint>;
    gas_limit: SSZCoder<number | bigint>;
    gas_used: SSZCoder<number | bigint>;
    timestamp: SSZCoder<number | bigint>;
    extra_data: ByteListType;
    base_fee_per_gas: SSZCoder<number | bigint>;
    block_hash: ByteVectorType;
    transactions_root: ByteVectorType;
    withdrawals_root: ByteVectorType;
}>;
export declare const CapellaBeaconBlock: ContainerCoder<{
    slot: SSZCoder<number | bigint>;
    proposer_index: SSZCoder<number | bigint>;
    parent_root: ByteVectorType;
    state_root: ByteVectorType;
    body: ContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            attestation_2: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attestations: ListType<{
            aggregation_bits: boolean[];
            data: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        execution_payload: ContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
            withdrawals_root: ByteVectorType;
        }>;
        bls_to_execution_changes: ListType<{
            message: {
                validator_index: number | bigint;
                from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                to_execution_address: Uint8Array<ArrayBufferLike>;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
}>;
export declare const CapellaSignedBeaconBlock: ContainerCoder<{
    message: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body: ContainerCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                attestation_2: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attestations: ListType<{
                aggregation_bits: boolean[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            sync_aggregate: ContainerCoder<{
                sync_committee_bits: BitVectorType;
                sync_committee_signature: ByteVectorType;
            }>;
            execution_payload: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
                withdrawals_root: ByteVectorType;
            }>;
            bls_to_execution_changes: ListType<{
                message: {
                    validator_index: number | bigint;
                    from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                    to_execution_address: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
        }>;
    }>;
    signature: ByteVectorType;
}>;
export declare const CapellaBeaconState: ContainerCoder<{
    genesis_time: SSZCoder<number | bigint>;
    genesis_validators_root: ByteVectorType;
    slot: SSZCoder<number | bigint>;
    fork: ContainerCoder<{
        previous_version: ByteVectorType;
        current_version: ByteVectorType;
        epoch: SSZCoder<number | bigint>;
    }>;
    latest_block_header: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body_root: ByteVectorType;
    }>;
    block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
    eth1_data: ContainerCoder<{
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
    }>;
    eth1_data_votes: ListType<{
        deposit_root: Uint8Array<ArrayBufferLike>;
        deposit_count: number | bigint;
        block_hash: Uint8Array<ArrayBufferLike>;
    }>;
    eth1_deposit_index: SSZCoder<number | bigint>;
    validators: ListType<{
        pubkey: Uint8Array<ArrayBufferLike>;
        withdrawal_credentials: Uint8Array<ArrayBufferLike>;
        effective_balance: number | bigint;
        slashed: boolean;
        activation_eligibility_epoch: number | bigint;
        activation_epoch: number | bigint;
        exit_epoch: number | bigint;
        withdrawable_epoch: number | bigint;
    }>;
    balances: ListType<number | bigint>;
    randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
    slashings: VectorType<number | bigint>;
    previous_epoch_participation: ListType<number | bigint>;
    current_epoch_participation: ListType<number | bigint>;
    justification_bits: BitVectorType;
    previous_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    current_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    finalized_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    inactivity_scores: ListType<number | bigint>;
    current_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    next_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    latest_execution_payload_header: ContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions_root: ByteVectorType;
        withdrawals_root: ByteVectorType;
    }>;
    next_withdrawal_index: SSZCoder<number | bigint>;
    next_withdrawal_validator_index: SSZCoder<number | bigint>;
    historical_summaries: ListType<{
        block_summary_root: Uint8Array<ArrayBufferLike>;
        state_summary_root: Uint8Array<ArrayBufferLike>;
    }>;
}>;
/** Bellatrix Types */
export declare const BellatrixExecutionPayloadHeader: ContainerCoder<{
    parent_hash: ByteVectorType;
    fee_recipient: ByteVectorType;
    state_root: ByteVectorType;
    receipts_root: ByteVectorType;
    logs_bloom: ByteVectorType;
    prev_randao: ByteVectorType;
    block_number: SSZCoder<number | bigint>;
    gas_limit: SSZCoder<number | bigint>;
    gas_used: SSZCoder<number | bigint>;
    timestamp: SSZCoder<number | bigint>;
    extra_data: ByteListType;
    base_fee_per_gas: SSZCoder<number | bigint>;
    block_hash: ByteVectorType;
    transactions_root: ByteVectorType;
}>;
export declare const BellatrixBeaconBlock: ContainerCoder<{
    slot: SSZCoder<number | bigint>;
    proposer_index: SSZCoder<number | bigint>;
    parent_root: ByteVectorType;
    state_root: ByteVectorType;
    body: ContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            attestation_2: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attestations: ListType<{
            aggregation_bits: boolean[];
            data: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
        execution_payload: ContainerCoder<{
            parent_hash: ByteVectorType;
            fee_recipient: ByteVectorType;
            state_root: ByteVectorType;
            receipts_root: ByteVectorType;
            logs_bloom: ByteVectorType;
            prev_randao: ByteVectorType;
            block_number: SSZCoder<number | bigint>;
            gas_limit: SSZCoder<number | bigint>;
            gas_used: SSZCoder<number | bigint>;
            timestamp: SSZCoder<number | bigint>;
            extra_data: ByteListType;
            base_fee_per_gas: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
            transactions_root: ByteVectorType;
        }>;
        bls_to_execution_changes: ListType<{
            message: {
                validator_index: number | bigint;
                from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                to_execution_address: Uint8Array<ArrayBufferLike>;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
}>;
export declare const BellatrixSignedBeaconBlock: ContainerCoder<{
    message: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body: ContainerCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                attestation_2: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attestations: ListType<{
                aggregation_bits: boolean[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            sync_aggregate: ContainerCoder<{
                sync_committee_bits: BitVectorType;
                sync_committee_signature: ByteVectorType;
            }>;
            execution_payload: ContainerCoder<{
                parent_hash: ByteVectorType;
                fee_recipient: ByteVectorType;
                state_root: ByteVectorType;
                receipts_root: ByteVectorType;
                logs_bloom: ByteVectorType;
                prev_randao: ByteVectorType;
                block_number: SSZCoder<number | bigint>;
                gas_limit: SSZCoder<number | bigint>;
                gas_used: SSZCoder<number | bigint>;
                timestamp: SSZCoder<number | bigint>;
                extra_data: ByteListType;
                base_fee_per_gas: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
                transactions_root: ByteVectorType;
            }>;
            bls_to_execution_changes: ListType<{
                message: {
                    validator_index: number | bigint;
                    from_bls_pubkey: Uint8Array<ArrayBufferLike>;
                    to_execution_address: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
        }>;
    }>;
    signature: ByteVectorType;
}>;
export declare const BellatrixBeaconState: ContainerCoder<{
    genesis_time: SSZCoder<number | bigint>;
    genesis_validators_root: ByteVectorType;
    slot: SSZCoder<number | bigint>;
    fork: ContainerCoder<{
        previous_version: ByteVectorType;
        current_version: ByteVectorType;
        epoch: SSZCoder<number | bigint>;
    }>;
    latest_block_header: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body_root: ByteVectorType;
    }>;
    block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
    eth1_data: ContainerCoder<{
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
    }>;
    eth1_data_votes: ListType<{
        deposit_root: Uint8Array<ArrayBufferLike>;
        deposit_count: number | bigint;
        block_hash: Uint8Array<ArrayBufferLike>;
    }>;
    eth1_deposit_index: SSZCoder<number | bigint>;
    validators: ListType<{
        pubkey: Uint8Array<ArrayBufferLike>;
        withdrawal_credentials: Uint8Array<ArrayBufferLike>;
        effective_balance: number | bigint;
        slashed: boolean;
        activation_eligibility_epoch: number | bigint;
        activation_epoch: number | bigint;
        exit_epoch: number | bigint;
        withdrawable_epoch: number | bigint;
    }>;
    balances: ListType<number | bigint>;
    randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
    slashings: VectorType<number | bigint>;
    previous_epoch_participation: ListType<number | bigint>;
    current_epoch_participation: ListType<number | bigint>;
    justification_bits: BitVectorType;
    previous_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    current_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    finalized_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    inactivity_scores: ListType<number | bigint>;
    current_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    next_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    latest_execution_payload_header: ContainerCoder<{
        parent_hash: ByteVectorType;
        fee_recipient: ByteVectorType;
        state_root: ByteVectorType;
        receipts_root: ByteVectorType;
        logs_bloom: ByteVectorType;
        prev_randao: ByteVectorType;
        block_number: SSZCoder<number | bigint>;
        gas_limit: SSZCoder<number | bigint>;
        gas_used: SSZCoder<number | bigint>;
        timestamp: SSZCoder<number | bigint>;
        extra_data: ByteListType;
        base_fee_per_gas: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
        transactions_root: ByteVectorType;
    }>;
}>;
export declare const AltairBeaconBlock: ContainerCoder<{
    slot: SSZCoder<number | bigint>;
    proposer_index: SSZCoder<number | bigint>;
    parent_root: ByteVectorType;
    state_root: ByteVectorType;
    body: ContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            attestation_2: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attestations: ListType<{
            aggregation_bits: boolean[];
            data: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        sync_aggregate: ContainerCoder<{
            sync_committee_bits: BitVectorType;
            sync_committee_signature: ByteVectorType;
        }>;
    }>;
}>;
export declare const AltairSignedBeaconBlock: ContainerCoder<{
    message: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body: ContainerCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                attestation_2: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attestations: ListType<{
                aggregation_bits: boolean[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            sync_aggregate: ContainerCoder<{
                sync_committee_bits: BitVectorType;
                sync_committee_signature: ByteVectorType;
            }>;
        }>;
    }>;
    signature: ByteVectorType;
}>;
export declare const AltairBeaconState: ContainerCoder<{
    genesis_time: SSZCoder<number | bigint>;
    genesis_validators_root: ByteVectorType;
    slot: SSZCoder<number | bigint>;
    fork: ContainerCoder<{
        previous_version: ByteVectorType;
        current_version: ByteVectorType;
        epoch: SSZCoder<number | bigint>;
    }>;
    latest_block_header: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body_root: ByteVectorType;
    }>;
    block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
    eth1_data: ContainerCoder<{
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
    }>;
    eth1_data_votes: ListType<{
        deposit_root: Uint8Array<ArrayBufferLike>;
        deposit_count: number | bigint;
        block_hash: Uint8Array<ArrayBufferLike>;
    }>;
    eth1_deposit_index: SSZCoder<number | bigint>;
    validators: ListType<{
        pubkey: Uint8Array<ArrayBufferLike>;
        withdrawal_credentials: Uint8Array<ArrayBufferLike>;
        effective_balance: number | bigint;
        slashed: boolean;
        activation_eligibility_epoch: number | bigint;
        activation_epoch: number | bigint;
        exit_epoch: number | bigint;
        withdrawable_epoch: number | bigint;
    }>;
    balances: ListType<number | bigint>;
    randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
    slashings: VectorType<number | bigint>;
    previous_epoch_participation: ListType<number | bigint>;
    current_epoch_participation: ListType<number | bigint>;
    justification_bits: BitVectorType;
    previous_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    current_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    finalized_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    inactivity_scores: ListType<number | bigint>;
    current_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
    next_sync_committee: ContainerCoder<{
        pubkeys: VectorType<Uint8Array<ArrayBufferLike>>;
        aggregate_pubkey: ByteVectorType;
    }>;
}>;
export declare const Phase0BeaconBlock: ContainerCoder<{
    slot: SSZCoder<number | bigint>;
    proposer_index: SSZCoder<number | bigint>;
    parent_root: ByteVectorType;
    state_root: ByteVectorType;
    body: ContainerCoder<{
        randao_reveal: ByteVectorType;
        eth1_data: ContainerCoder<{
            deposit_root: ByteVectorType;
            deposit_count: SSZCoder<number | bigint>;
            block_hash: ByteVectorType;
        }>;
        graffiti: ByteVectorType;
        proposer_slashings: ListType<{
            signed_header_1: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            signed_header_2: {
                message: {
                    slot: number | bigint;
                    proposer_index: number | bigint;
                    parent_root: Uint8Array<ArrayBufferLike>;
                    state_root: Uint8Array<ArrayBufferLike>;
                    body_root: Uint8Array<ArrayBufferLike>;
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attester_slashings: ListType<{
            attestation_1: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
            attestation_2: {
                attesting_indices: (number | bigint)[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        attestations: ListType<{
            aggregation_bits: boolean[];
            data: {
                slot: number | bigint;
                index: number | bigint;
                beacon_block_root: Uint8Array<ArrayBufferLike>;
                source: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
                target: {
                    epoch: number | bigint;
                    root: Uint8Array<ArrayBufferLike>;
                };
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
        deposits: ListType<{
            proof: Uint8Array<ArrayBufferLike>[];
            data: {
                pubkey: Uint8Array<ArrayBufferLike>;
                withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                amount: number | bigint;
                signature: Uint8Array<ArrayBufferLike>;
            };
        }>;
        voluntary_exits: ListType<{
            message: {
                epoch: number | bigint;
                validator_index: number | bigint;
            };
            signature: Uint8Array<ArrayBufferLike>;
        }>;
    }>;
}>;
export declare const Phase0SignedBeaconBlock: ContainerCoder<{
    message: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body: ContainerCoder<{
            randao_reveal: ByteVectorType;
            eth1_data: ContainerCoder<{
                deposit_root: ByteVectorType;
                deposit_count: SSZCoder<number | bigint>;
                block_hash: ByteVectorType;
            }>;
            graffiti: ByteVectorType;
            proposer_slashings: ListType<{
                signed_header_1: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                signed_header_2: {
                    message: {
                        slot: number | bigint;
                        proposer_index: number | bigint;
                        parent_root: Uint8Array<ArrayBufferLike>;
                        state_root: Uint8Array<ArrayBufferLike>;
                        body_root: Uint8Array<ArrayBufferLike>;
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attester_slashings: ListType<{
                attestation_1: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
                attestation_2: {
                    attesting_indices: (number | bigint)[];
                    data: {
                        slot: number | bigint;
                        index: number | bigint;
                        beacon_block_root: Uint8Array<ArrayBufferLike>;
                        source: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                        target: {
                            epoch: number | bigint;
                            root: Uint8Array<ArrayBufferLike>;
                        };
                    };
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            attestations: ListType<{
                aggregation_bits: boolean[];
                data: {
                    slot: number | bigint;
                    index: number | bigint;
                    beacon_block_root: Uint8Array<ArrayBufferLike>;
                    source: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                    target: {
                        epoch: number | bigint;
                        root: Uint8Array<ArrayBufferLike>;
                    };
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
            deposits: ListType<{
                proof: Uint8Array<ArrayBufferLike>[];
                data: {
                    pubkey: Uint8Array<ArrayBufferLike>;
                    withdrawal_credentials: Uint8Array<ArrayBufferLike>;
                    amount: number | bigint;
                    signature: Uint8Array<ArrayBufferLike>;
                };
            }>;
            voluntary_exits: ListType<{
                message: {
                    epoch: number | bigint;
                    validator_index: number | bigint;
                };
                signature: Uint8Array<ArrayBufferLike>;
            }>;
        }>;
    }>;
    signature: ByteVectorType;
}>;
export declare const Phase0BeaconState: ContainerCoder<{
    genesis_time: SSZCoder<number | bigint>;
    genesis_validators_root: ByteVectorType;
    slot: SSZCoder<number | bigint>;
    fork: ContainerCoder<{
        previous_version: ByteVectorType;
        current_version: ByteVectorType;
        epoch: SSZCoder<number | bigint>;
    }>;
    latest_block_header: ContainerCoder<{
        slot: SSZCoder<number | bigint>;
        proposer_index: SSZCoder<number | bigint>;
        parent_root: ByteVectorType;
        state_root: ByteVectorType;
        body_root: ByteVectorType;
    }>;
    block_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    state_roots: VectorType<Uint8Array<ArrayBufferLike>>;
    historical_roots: ListType<Uint8Array<ArrayBufferLike>>;
    eth1_data: ContainerCoder<{
        deposit_root: ByteVectorType;
        deposit_count: SSZCoder<number | bigint>;
        block_hash: ByteVectorType;
    }>;
    eth1_data_votes: ListType<{
        deposit_root: Uint8Array<ArrayBufferLike>;
        deposit_count: number | bigint;
        block_hash: Uint8Array<ArrayBufferLike>;
    }>;
    eth1_deposit_index: SSZCoder<number | bigint>;
    validators: ListType<{
        pubkey: Uint8Array<ArrayBufferLike>;
        withdrawal_credentials: Uint8Array<ArrayBufferLike>;
        effective_balance: number | bigint;
        slashed: boolean;
        activation_eligibility_epoch: number | bigint;
        activation_epoch: number | bigint;
        exit_epoch: number | bigint;
        withdrawable_epoch: number | bigint;
    }>;
    balances: ListType<number | bigint>;
    randao_mixes: VectorType<Uint8Array<ArrayBufferLike>>;
    slashings: VectorType<number | bigint>;
    previous_epoch_participation: ListType<number | bigint>;
    current_epoch_participation: ListType<number | bigint>;
    justification_bits: BitVectorType;
    previous_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    current_justified_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
    finalized_checkpoint: ContainerCoder<{
        epoch: SSZCoder<number | bigint>;
        root: ByteVectorType;
    }>;
}>;
export {};
//# sourceMappingURL=ssz.d.ts.map