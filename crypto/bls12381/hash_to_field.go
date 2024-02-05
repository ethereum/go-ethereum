package bls12381

import (
	"crypto/sha256"
	"errors"
)

func hashToFpXMDSHA256(msg []byte, domain []byte, count int) ([]*fe, error) {
	randBytes, err := expandMsgSHA256XMD(msg, domain, count*64)
	if err != nil {
		return nil, err
	}
	els := make([]*fe, count)
	for i := 0; i < count; i++ {
		els[i], err = from64Bytes(randBytes[i*64 : (i+1)*64])
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func expandMsgSHA256XMD(msg []byte, domain []byte, outLen int) ([]byte, error) {
	h := sha256.New()
	domainLen := uint8(len(domain))
	if domainLen > 255 {
		return nil, errors.New("invalid domain length")
	}
	// DST_prime = DST || I2OSP(len(DST), 1)
	// b_0 = H(Z_pad || msg || l_i_b_str || I2OSP(0, 1) || DST_prime)
	_, _ = h.Write(make([]byte, h.BlockSize()))
	_, _ = h.Write(msg)
	_, _ = h.Write([]byte{uint8(outLen >> 8), uint8(outLen)})
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(domain)
	_, _ = h.Write([]byte{domainLen})
	b0 := h.Sum(nil)

	// b_1 = H(b_0 || I2OSP(1, 1) || DST_prime)
	h.Reset()
	_, _ = h.Write(b0)
	_, _ = h.Write([]byte{1})
	_, _ = h.Write(domain)
	_, _ = h.Write([]byte{domainLen})
	b1 := h.Sum(nil)

	// b_i = H(strxor(b_0, b_(i - 1)) || I2OSP(i, 1) || DST_prime)
	ell := (outLen + h.Size() - 1) / h.Size()
	bi := b1
	out := make([]byte, outLen)
	for i := 1; i < ell; i++ {
		h.Reset()
		// b_i = H(strxor(b_0, b_(i - 1)) || I2OSP(i, 1) || DST_prime)
		tmp := make([]byte, h.Size())
		for j := 0; j < h.Size(); j++ {
			tmp[j] = b0[j] ^ bi[j]
		}
		_, _ = h.Write(tmp)
		_, _ = h.Write([]byte{1 + uint8(i)})
		_, _ = h.Write(domain)
		_, _ = h.Write([]byte{domainLen})

		// b_1 || ... || b_(ell - 1)
		copy(out[(i-1)*h.Size():i*h.Size()], bi[:])
		bi = h.Sum(nil)
	}
	// b_ell
	copy(out[(ell-1)*h.Size():], bi[:])
	return out[:outLen], nil
}
