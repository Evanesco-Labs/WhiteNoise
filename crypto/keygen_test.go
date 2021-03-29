package crypto

import (
	"crypto/rand"
	"fmt"
	"github.com/magiconair/properties/assert"
	"github.com/mr-tron/base58"
	ct "github.com/nspcc-dev/neofs-crypto"
	"testing"
)

func TestMarshallECDSAPublicKey(t *testing.T) {
	priv, err := GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey
	compress := ct.MarshalPublicKey(&pub)
	keyString := base58.Encode(compress)
	fmt.Println(keyString)
	pk := ct.UnmarshalPublicKey(compress)
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
	pub = *ecdsaPK
	compress = ct.MarshalPublicKey(&pub)
	keyStringC := base58.Encode(compress)
	assert.Equal(t, keyString, keyStringC)
}
