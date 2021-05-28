package crypto

import (
	"crypto/rand"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestEd25519Generate(t *testing.T) {

	priv, _, err := GenerateEd25519KeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	p2pPriv, p2pPub, err := P2PKeypairFromEd25519(priv)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, p2pPriv.GetPublic().Equals(p2pPub), true)

}
