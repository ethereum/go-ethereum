// Copyright 2025 go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pbt

// Copy the values here so as to avoid an import cycle.
const (
	opPUSH1  = byte(0x60)
	opPUSH32 = byte(0x7f)

	codeChunkSize     = 32 // size of a code chunk, metadata byte included
	codeChunkDataSize = 31 // number of code bytes in a chunk
)

// chunkifyCode generates the chunked version of an array representing EVM bytecode.
//
// The code is divided into 32-byte chunks, where each chunk contains:
//   - Byte 0: Metadata byte indicating the number of leading bytes that are PUSHDATA (0-31)
//   - Bytes 1-31: Actual code bytes
//
// This format enables stateless clients to validate jump destinations within a chunk
// without requiring additional context. When a PUSH instruction's data spans multiple
// chunks, the metadata byte tells us how many bytes at the start of the chunk are
// part of the previous chunk's PUSH instruction data.
//
// For example:
//   - If a chunk starts with regular code: metadata byte = 0
//   - If a PUSH32 instruction starts at byte 30 of chunk N:
//   - Chunk N: normal, contains PUSH32 opcode + 1 byte of data
//   - Chunk N+1: metadata = 31 (entire chunk is PUSH data)
//   - Chunk N+2: metadata = 1 (first byte is PUSH data, then normal code resumes)
func chunkifyCode(code []byte) []byte {
	var (
		chunkOffset = 0 // offset in the chunk
		chunkCount  = len(code) / codeChunkDataSize
		codeOffset  = 0 // offset in the code
	)
	if len(code)%codeChunkDataSize != 0 {
		chunkCount++
	}
	chunks := make([]byte, chunkCount*codeChunkSize)
	for i := range chunkCount {
		// number of bytes to copy, codeChunkDataSize unless the end of the code has been reached.
		end := min(len(code), codeChunkDataSize*(i+1))
		copy(chunks[i*codeChunkSize+1:], code[codeChunkDataSize*i:end]) // copy the code itself

		// chunk offset = taken from the last chunk.
		if chunkOffset > codeChunkDataSize {
			// skip offset calculation if push data covers the whole chunk
			chunks[i*codeChunkSize] = codeChunkDataSize
			chunkOffset = 1
			continue
		}
		chunks[codeChunkSize*i] = byte(chunkOffset)
		chunkOffset = 0

		// Check each instruction and update the offset it should be 0 unless
		// a PUSH-N overflows.
		for ; codeOffset < end; codeOffset++ {
			if code[codeOffset] >= opPUSH1 && code[codeOffset] <= opPUSH32 {
				codeOffset += int(code[codeOffset] - opPUSH1 + 1)
				if codeOffset+1 >= codeChunkDataSize*(i+1) {
					codeOffset++
					chunkOffset = codeOffset - codeChunkDataSize*(i+1)
					break
				}
			}
		}
	}
	return chunks
}
