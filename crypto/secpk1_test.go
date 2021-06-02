package crypto

import (
	"crypto/rand"
	"fmt"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestMarshallSecp256k1PublicKey(t *testing.T) {
	_, pub, err := GenerateSecp256k1KeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pub bytes", pub.Bytes())
	fmt.Println("len", len(pub.Bytes()))
}

func TestECIESSecp(t *testing.T) {
	message := []byte("hello ecies")

	priv, pub, err := GenerateSecp256k1KeyPair(rand.Reader)
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
