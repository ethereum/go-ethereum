func symEncrypt(rand io.Reader, params *ECIESParams, key, m []byte) (ct []byte, err error) {
	c, err := params.Cipher(key)
	if err != nil {
		return
	}

	iv, err := generateIV(params, rand)
	if err != nil {
		return
	}
	ctr := cipher.NewCTR(c, iv)

	ct = make([]byte, len(m)+params.BlockSize)
	copy(ct, iv)
	ctr.XORKeyStream(ct[params.BlockSize:], m)
	return
}

// symDecrypt carries out CTR decryption using the block cipher specified in
// the parameters
func symDecrypt(params *ECIESParams, key, ct []byte) (m []byte, err error) {
	c, err := params.Cipher(key)
	if err != nil {
		return
	}

	ctr := cipher.NewCTR(c, ct[:params.BlockSize])

	m = make([]byte, len(ct)-params.BlockSize)
	ctr.XORKeyStream(m, ct[params.BlockSize:])
	return
}

// Encrypt encrypts a message using ECIES as specified in SEC 1, 5.1.
//
// s1 and s2 contain shared information that is not part of the resulting
// ciphertext. s1 is fed into key derivation, s2 is fed into the MAC. If the
// shared information parameters aren't being used, they should be nil.
func Encrypt(rand io.Reader, pub *PublicKey, m, s1, s2 []byte) (ct []byte, err error) {
	params, err := pubkeyParams(pub)
	if err != nil {
		return nil, err
	}

	R, err := GenerateKey(rand, pub.Curve, params)
	if err != nil {
		return nil, err
	}

	z, err := R.GenerateShared(pub, params.KeyLen, params.KeyLen)
	if err != nil {
		return nil, err
	}

	hash := params.Hash()
	Ke, Km := deriveKeys(hash, z, s1, params.KeyLen)

	em, err := symEncrypt(rand, params, Ke, m)
	if err != nil || len(em) <= params.BlockSize {
		return nil, err
	}

	d := messageTag(params.Hash, Km, em, s2)

	Rb := elliptic.Marshal(pub.Curve, R.PublicKey.X, R.PublicKey.Y)
	ct = make([]byte, len(Rb)+len(em)+len(d))
	copy(ct, Rb)
	copy(ct[len(Rb):], em)
	copy(ct[len(Rb)+len(em):], d)
	return ct, nil
}

// Decrypt decrypts an ECIES ciphertext.
func (prv *PrivateKey) Decrypt(c, s1, s2 []byte) (m []byte, err error) {
	if len(c) == 0 {
		return nil, ErrInvalidMessage
	}
	params, err := pubkeyParams(&prv.PublicKey)
	if err != nil {
		return nil, err
	}

	hash := params.Hash()

	var (
		rLen   int
		hLen   int = hash.Size()
		mStart int
		mEnd   int
	)

	switch c[0] {
	case 2, 3, 4:
		rLen = (prv.PublicKey.Curve.Params().BitSize + 7) / 4
		if len(c) < (rLen + hLen + 1) {
			return nil, ErrInvalidMessage
		}
	default:
		return nil, ErrInvalidPublicKey
	}

	mStart = rLen
	mEnd = len(c) - hLen

	R := new(PublicKey)
	R.Curve = prv.PublicKey.Curve
	R.X, R.Y = elliptic.Unmarshal(R.Curve, c[:rLen])
	if R.X == nil {
		return nil, ErrInvalidPublicKey
	}
	if !R.Curve.IsOnCurve(R.X, R.Y) {
		return nil, ErrInvalidCurve
	}

	z, err := prv.GenerateShared(R, params.KeyLen, params.KeyLen)
	if err != nil {
		return nil, err
	}
	Ke, Km := deriveKeys(hash, z, s1, params.KeyLen)

	d := messageTag(params.Hash, Km, c[mStart:mEnd], s2)
	if subtle.ConstantTimeCompare(c[mEnd:], d) != 1 {
		return nil, ErrInvalidMessage
	}

	return symDecrypt(params, Ke, c[mStart:mEnd])
}
