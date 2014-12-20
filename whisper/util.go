package whisper

import "github.com/ethereum/go-ethereum/crypto"

func hashTopic(topic []byte) []byte {
	return crypto.Sha3(topic)[:4]
}

// NOTE this isn't DRY, but I don't want to iterate twice.

// Returns a formatted topics byte slice.
// data: unformatted data (e.g., no hashes needed)
func Topics(data [][]byte) [][]byte {
	d := make([][]byte, len(data))
	for i, byts := range data {
		d[i] = hashTopic(byts)
	}
	return d
}

func TopicsFromString(data []string) [][]byte {
	d := make([][]byte, len(data))
	for i, str := range data {
		d[i] = hashTopic([]byte(str))
	}
	return d
}
