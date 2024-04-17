package state

import (
	"errors"
	"fmt"

	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

var _ common.SSZObj = (*Nibbles)(nil)

type Nibbles struct {
	Nibbles []byte
}

func (n *Nibbles) Serialize(w *codec.EncodingWriter) error {
	if len(n.Nibbles)%2 == 0 {
		err := w.WriteByte(0)
		if err != nil {
			return err
		}

		for i := 0; i < len(n.Nibbles); i += 2 {
			err = w.WriteByte(n.Nibbles[i]<<4 | n.Nibbles[i+1])
			if err != nil {
				return err
			}
		}
	} else {
		err := w.WriteByte(0x10 | n.Nibbles[0])
		if err != nil {
			return err
		}

		for i := 1; i < len(n.Nibbles); i += 2 {
			err = w.WriteByte(n.Nibbles[i]<<4 | n.Nibbles[i+1])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Nibbles) ByteLength() uint64 {
	return uint64(len(n.Nibbles)/2 + 1)
}

func (n *Nibbles) FixedLength() uint64 {
	return 0
}

func (n *Nibbles) Deserialize(dr *codec.DecodingReader) error {
	firstByte, err := dr.ReadByte()
	if err != nil {
		return err
	}

	packedNibbles := make([]byte, dr.Scope())
	_, err = dr.Read(packedNibbles)
	if err != nil {
		return err
	}
	flag, first := unpackNibblePair(firstByte)
	nibbles := make([]byte, 1+2*len(packedNibbles))

	if flag == 0 {
		if first != 0 {
			return fmt.Errorf("nibbles: The lowest 4 bits of the first byte must be 0, but was: %x", first)
		}
	} else if flag == 1 {
		nibbles = append(nibbles, first)
	} else {
		return fmt.Errorf("nibbles: The highest 4 bits of the first byte must be 0 or 1, but was: %x", flag)
	}

	for i, b := range packedNibbles {
		left, right := unpackNibblePair(b)
		nibbles[1+2*i] = left
		nibbles[1+2*i+1] = right
	}

	unpackedNibbles, err := FromUnpackedNibbles(nibbles)
	if err != nil {
		return err
	}

	*n = *unpackedNibbles
	return nil
}

func (n *Nibbles) HashTreeRoot(h tree.HashFn) tree.Root {
	//TODO implement me
	panic("implement me")
}

func FromUnpackedNibbles(nibbles []byte) (*Nibbles, error) {
	if len(nibbles) > 64 {
		return nil, errors.New("too many nibbles")
	}

	for _, nibble := range nibbles {
		if nibble > 0xf {
			return nil, errors.New("nibble out of range")
		}
	}

	return &Nibbles{Nibbles: nibbles}, nil
}

func unpackNibblePair(pair byte) (byte, byte) {
	return pair >> 4, pair & 0xf
}
