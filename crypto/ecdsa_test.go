package crypto

import (
	"crypto/rand"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestMarshal(t *testing.T) {
	_, pub, _ := GenerateECDSAKeyPair(rand.Reader)
	public := pub.(ECDSAPublicKey)
	data, err := MarshallECDSAPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	_, err = UnMarshallECDSAPublicKey(data)
	if err != nil {
		t.Fatal(err)
	}
}

func TestECIESECDSA(t *testing.T) {
	message := []byte("hello ecies")

	priv, pub, err := GenerateECDSAKeyPair(rand.Reader)
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
