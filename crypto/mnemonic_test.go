package crypto

import (
	"testing"
)

func TestMnDecode(t *testing.T) {
	words := []string{
		"ink",
		"balance",
		"gain",
		"fear",
		"happen",
		"melt",
		"mom",
		"surface",
		"stir",
		"bottle",
		"unseen",
		"expression",
		"important",
		"curl",
		"grant",
		"fairy",
		"across",
		"back",
		"figure",
		"breast",
		"nobody",
		"scratch",
		"worry",
		"yesterday",
	}
	encode := "c61d43dc5bb7a4e754d111dae8105b6f25356492df5e50ecb33b858d94f8c338"
	result := MnemonicDecode(words)
	if encode != result {
		t.Error("We expected", encode, "got", result, "instead")
	}
}
func TestMnEncode(t *testing.T) {
	encode := "c61d43dc5bb7a4e754d111dae8105b6f25356492df5e50ecb33b858d94f8c338"
	result := []string{
		"ink",
		"balance",
		"gain",
		"fear",
		"happen",
		"melt",
		"mom",
		"surface",
		"stir",
		"bottle",
		"unseen",
		"expression",
		"important",
		"curl",
		"grant",
		"fairy",
		"across",
		"back",
		"figure",
		"breast",
		"nobody",
		"scratch",
		"worry",
		"yesterday",
	}
	words := MnemonicEncode(encode)
	for i, word := range words {
		if word != result[i] {
			t.Error("Mnenonic does not match:", words, result)
		}
	}
}
