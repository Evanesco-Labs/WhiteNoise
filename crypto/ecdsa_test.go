package crypto

import (
	"crypto/rand"
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
