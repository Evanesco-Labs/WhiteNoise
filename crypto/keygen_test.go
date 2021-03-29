package crypto

import (
	"crypto/rand"
	"github.com/magiconair/properties/assert"
	"github.com/mr-tron/base58"
	"testing"
)

func TestMarshallECDSAPublicKey(t *testing.T) {
	priv, err := GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey
	compress, err := MarshallECDSAPublicKey(&pub)
	if err != nil {
		t.Fatal(err)
	}
	keyString := base58.Encode(compress)

	pk, err := UnMarshallECDSAPublicKey(compress)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, pk.X.String(), pub.X.String())
	assert.Equal(t, pk.Y.String(), pub.Y.String())

	_, p2pPK, err := P2PKeypairFromECDSA(priv)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaPK, err := ECDSAPublicKeyFromP2PPK(p2pPK)
	if err != nil {
		t.Fatal(err)
	}

	compress, err = MarshallECDSAPublicKey(ecdsaPK)
	if err != nil {
		t.Fatal(err)
	}
	keyStringC := base58.Encode(compress)
	assert.Equal(t, keyString, keyStringC)
}
