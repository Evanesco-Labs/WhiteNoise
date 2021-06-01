package crypto

import (
	"crypto/rand"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestECIES(t *testing.T) {
	message := []byte("hello ecies")

	priv, pub, err := GenerateEd25519KeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cyphertext, err := pub.ECIESEncrypt(message, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := priv.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, plaintext, message)
}
