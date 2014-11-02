package rle

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	token             byte = 0xfe
	emptyShaToken          = 0xfd
	emptyListShaToken      = 0xfe
	tokenToken             = 0xff
)

var empty = crypto.Sha3([]byte(""))
var emptyList = crypto.Sha3([]byte{0x80})

func Decompress(dat []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	for i := 0; i < len(dat); i++ {
		if dat[i] == token {
			if i+1 < len(dat) {
				switch dat[i+1] {
				case emptyShaToken:
					buf.Write(empty)
				case emptyListShaToken:
					buf.Write(emptyList)
				case tokenToken:
					buf.WriteByte(token)
				default:
					buf.Write(make([]byte, int(dat[i+1]-2)))
				}
				i++
			} else {
				return nil, errors.New("error reading bytes. token encountered without proceeding bytes")
			}
		}
	}

	return buf.Bytes(), nil
}

func Compress(dat []byte) []byte {
	buf := new(bytes.Buffer)

	for i := 0; i < len(dat); i++ {
		if dat[i] == token {
			buf.Write([]byte{token, tokenToken})
		} else if i+1 < len(dat) {
			if dat[i] == 0x0 && dat[i+1] == 0x0 {
				j := 0
				for j <= 254 && i+j < len(dat) {
					if dat[i+j] != 0 {
						break
					}
					j++
				}
				buf.Write([]byte{token, byte(j + 2)})
				i += (j - 1)
			} else if len(dat[i:]) >= 32 {
				if dat[i] == empty[0] && bytes.Compare(dat[i:i+32], empty) == 0 {
					buf.Write([]byte{token, emptyShaToken})
					i += 31
				} else if dat[i] == emptyList[0] && bytes.Compare(dat[i:i+32], emptyList) == 0 {
					buf.Write([]byte{token, emptyListShaToken})
					i += 31
				} else {
					buf.WriteByte(dat[i])
				}
			} else {
				buf.WriteByte(dat[i])
			}
		} else {
			buf.WriteByte(dat[i])
		}
	}

	return buf.Bytes()
}
