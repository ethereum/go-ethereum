package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"
)

// Константы получены подписантом d0_sign.py для tx: chainId 1337, nonce 0,
// gasPrice 1 gwei, gas 21000, to 0x7e5f…95bdf, value 0.5 ETH.
const (
	testHash  = "dab7dc9f25247828219ba160c29cfeaa896bd943fb60f5b9308058f064870d2b"
	testSig   = "dbd5883a2180ae66173ea71181926198e64067fba46c956629409727b5d598e85366cb732b41a51421e448ab7de30eb5b6656a7c5b215b01651bc2a9dda6e51500" // d = 0
	testSigD1 = "dbd5883a2180ae66173ea71181926198e64067fba46c956629409727b5d598e815df9c7a004459ba51b666c1bda7c714f35c99d2e2adc1cdd81697a7bc4133b700" // d = 1
)

func allZero(b []byte) bool { return bytes.Equal(b, make([]byte, len(b))) }

// Подпись ключом d = 0: восстановление обязано дать 04||64 нуля и адрес 0x3f17…
func TestInfinityRecoveryZeroKey(t *testing.T) {
	hash, _ := hex.DecodeString(testHash)
	sig, _ := hex.DecodeString(testSig)

	pub, err := Ecrecover(hash, sig)
	if err != nil {
		t.Fatalf("Ecrecover: %v", err)
	}
	if len(pub) != 65 || pub[0] != 4 || !allZero(pub[1:]) {
		t.Fatalf("ожидали 04||64 нуля, получили %s", hex.EncodeToString(pub))
	}
	zeroKey := &ecdsa.PublicKey{Curve: S256(), X: new(big.Int), Y: new(big.Int)}
	if got := PubkeyToAddress(*zeroKey).Hex(); got != "0x3f17f1962B36e491b30A40b2405849e597Ba5FB5" {
		t.Fatalf("адрес нулевого ключа не совпал: %s", got)
	}

	// SigToPub — второй публичный путь, не через recoverPlain: не должен падать.
	npub, err := SigToPub(hash, sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	if npub.X == nil || npub.Y == nil || npub.X.Sign() != 0 || npub.Y.Sign() != 0 {
		t.Fatalf("SigToPub вернул не (0,0): %v %v", npub.X, npub.Y)
	}
}

// Обычная подпись (d = 1) не должна пострадать.
func TestNormalSignatureUnchanged(t *testing.T) {
	hash, _ := hex.DecodeString(testHash)
	sig, _ := hex.DecodeString(testSigD1)
	pub, err := Ecrecover(hash, sig)
	if err != nil {
		t.Fatalf("Ecrecover(d=1): %v", err)
	}
	if len(pub) != 65 || allZero(pub[1:]) {
		t.Fatalf("подпись d=1 принята за бесконечность: %s", hex.EncodeToString(pub))
	}
}

// Мусорная подпись (r = 5, s = 1) не должна «становиться бесконечностью».
func TestGarbageSignatureStaysInvalid(t *testing.T) {
	hash, _ := hex.DecodeString(testHash)
	sig := make([]byte, 65)
	sig[31] = 5 // r = 5
	sig[63] = 1 // s = 1
	if sigRecoversInfinity(hash, sig) {
		t.Fatalf("мусорная подпись распознана как бесконечность")
	}
}

// Проверка самого предиката: он обязан срабатывать именно на d = 0.
func TestInfinityPredicate(t *testing.T) {
	hash, _ := hex.DecodeString(testHash)
	sig0, _ := hex.DecodeString(testSig)
	sig1, _ := hex.DecodeString(testSigD1)
	if !sigRecoversInfinity(hash, sig0) {
		t.Fatalf("предикат не сработал на подписи d=0")
	}
	if sigRecoversInfinity(hash, sig1) {
		t.Fatalf("предикат ошибочно сработал на подписи d=1")
	}
}
