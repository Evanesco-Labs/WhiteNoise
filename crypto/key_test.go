package crypto

import (
	"crypto/rand"
	"fmt"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestMarshall(t *testing.T) {
	priv, pub, _ := GenerateKeyPair(Ed25519, rand.Reader)
	fmt.Println(len(pub.GetWhiteNoiseID()))
	fmt.Println(len(priv.Bytes()))

	sk, pk, _ := GenerateKeyPair(ECDSA, rand.Reader)
	fmt.Println(len(pk.GetWhiteNoiseID()))
	fmt.Println(len(sk.Bytes()))
}

func TestWhiteNoiseIDEd25519(t *testing.T) {
	priv, pub, _ := GenerateKeyPair(Ed25519, rand.Reader)
	id := pub.GetWhiteNoiseID().String()
	fmt.Println(id)
	message := []byte("hello whitenoise")

	whitenoiseId, err := WhiteNoiseIDfromString(id)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := whitenoiseId.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	cyphertext, err := pk.ECIESEncrypt(message, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := priv.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, plaintext, message)
}

func TestWhiteNoiseIDSecp(t *testing.T) {
	priv, pub, _ := GenerateKeyPair(Secpk1, rand.Reader)
	id := pub.GetWhiteNoiseID().String()
	fmt.Println(id)
	message := []byte("hello whitenoise")

	whitenoiseId, err := WhiteNoiseIDfromString(id)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := whitenoiseId.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	cyphertext, err := pk.ECIESEncrypt(message, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := priv.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, plaintext, message)
}

func TestWhiteNoiseIDECDSA(t *testing.T) {
	priv, pub, _ := GenerateKeyPair(ECDSA, rand.Reader)
	id := pub.GetWhiteNoiseID().String()
	fmt.Println(id)
	message := []byte("hello whitenoise")

	whitenoiseId, err := WhiteNoiseIDfromString(id)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := whitenoiseId.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	cyphertext, err := pk.ECIESEncrypt(message, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := priv.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, plaintext, message)
}

func TestWhiteNoiseID(t *testing.T) {
	priv, pub, _ := GenerateKeyPair(Ed25519, rand.Reader)
	id := pub.GetWhiteNoiseID().String()
	fmt.Println(id)
	message := []byte("hello whitenoise")

	whitenoiseId, err := WhiteNoiseIDfromString(id)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := whitenoiseId.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	cyphertext, err := pk.ECIESEncrypt(message, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := priv.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, plaintext, message)
}