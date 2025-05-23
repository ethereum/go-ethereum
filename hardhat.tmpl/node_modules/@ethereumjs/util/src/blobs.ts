import { sha256 } from 'ethereum-cryptography/sha256.js'

import { utf8ToBytes } from './bytes.js'

import type { Kzg } from './kzg.js'

/**
 * These utilities for constructing blobs are borrowed from https://github.com/Inphi/eip4844-interop.git
 */
const BYTES_PER_FIELD_ELEMENT = 32
const FIELD_ELEMENTS_PER_BLOB = 4096
const USEFUL_BYTES_PER_BLOB = 32 * FIELD_ELEMENTS_PER_BLOB
const MAX_BLOBS_PER_TX = 2
const MAX_USEFUL_BYTES_PER_TX = USEFUL_BYTES_PER_BLOB * MAX_BLOBS_PER_TX - 1
const BLOB_SIZE = BYTES_PER_FIELD_ELEMENT * FIELD_ELEMENTS_PER_BLOB

function get_padded(data: Uint8Array, blobs_len: number): Uint8Array {
  const pdata = new Uint8Array(blobs_len * USEFUL_BYTES_PER_BLOB).fill(0)
  pdata.set(data)
  pdata[data.byteLength] = 0x80
  return pdata
}

function get_blob(data: Uint8Array): Uint8Array {
  const blob = new Uint8Array(BLOB_SIZE)
  for (let i = 0; i < FIELD_ELEMENTS_PER_BLOB; i++) {
    const chunk = new Uint8Array(32)
    chunk.set(data.subarray(i * 31, (i + 1) * 31), 0)
    blob.set(chunk, i * 32)
  }

  return blob
}

export const getBlobs = (input: string) => {
  const data = utf8ToBytes(input)
  const len = data.byteLength
  if (len === 0) {
    throw Error('invalid blob data')
  }
  if (len > MAX_USEFUL_BYTES_PER_TX) {
    throw Error('blob data is too large')
  }

  const blobs_len = Math.ceil(len / USEFUL_BYTES_PER_BLOB)

  const pdata = get_padded(data, blobs_len)

  const blobs: Uint8Array[] = []
  for (let i = 0; i < blobs_len; i++) {
    const chunk = pdata.subarray(i * USEFUL_BYTES_PER_BLOB, (i + 1) * USEFUL_BYTES_PER_BLOB)
    const blob = get_blob(chunk)
    blobs.push(blob)
  }

  return blobs
}

export const blobsToCommitments = (kzg: Kzg, blobs: Uint8Array[]) => {
  const commitments: Uint8Array[] = []
  for (const blob of blobs) {
    commitments.push(kzg.blobToKzgCommitment(blob))
  }
  return commitments
}

export const blobsToProofs = (kzg: Kzg, blobs: Uint8Array[], commitments: Uint8Array[]) => {
  const proofs = blobs.map((blob, ctx) => kzg.computeBlobKzgProof(blob, commitments[ctx]))

  return proofs
}

/**
 * Converts a vector commitment for a given data blob to its versioned hash.  For 4844, this version
 * number will be 0x01 for KZG vector commitments but could be different if future vector commitment
 * types are introduced
 * @param commitment a vector commitment to a blob
 * @param blobCommitmentVersion the version number corresponding to the type of vector commitment
 * @returns a versioned hash corresponding to a given blob vector commitment
 */
export const computeVersionedHash = (commitment: Uint8Array, blobCommitmentVersion: number) => {
  const computedVersionedHash = new Uint8Array(32)
  computedVersionedHash.set([blobCommitmentVersion], 0)
  computedVersionedHash.set(sha256(commitment).subarray(1), 1)
  return computedVersionedHash
}

/**
 * Generate an array of versioned hashes from corresponding kzg commitments
 * @param commitments array of kzg commitments
 * @returns array of versioned hashes
 * Note: assumes KZG commitments (version 1 version hashes)
 */
export const commitmentsToVersionedHashes = (commitments: Uint8Array[]) => {
  const hashes: Uint8Array[] = []
  for (const commitment of commitments) {
    hashes.push(computeVersionedHash(commitment, 0x01))
  }
  return hashes
}
