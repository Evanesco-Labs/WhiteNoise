package account

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/magiconair/properties/assert"
	"testing"
	"github.com/Evanesco-Labs/WhiteNoise/crypto"
)

func TestAccountECDSA(t *testing.T) {
	acc, err := NewOneTimeAccount(crypto.ECDSA)
	if err != nil {
		t.Fatal(err)
	}
	idOri := acc.GetPublicKey().GetWhiteNoiseID()
	id, err := crypto.WhiteNoiseIDfromString(idOri.String())
	if err != nil {
		t.Fatal(err)
	}
	privP2P := acc.GetP2PPrivKey()
	idP2P, err := peer.IDFromPublicKey(privP2P.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	idP2P2, err := id.GetPeerID()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, idP2P.String(), idP2P2.String())

	msg := []byte("hello white noise")

	cyphertext, err := acc.pubKey.ECIESEncrypt(msg, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := acc.privKey.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, msg, plaintext)
}

func TestAccountEd25519(t *testing.T) {
	acc, err := NewOneTimeAccount(crypto.Ed25519)
	if err != nil {
		t.Fatal(err)
	}
	idOri := acc.GetPublicKey().GetWhiteNoiseID()
	id, err := crypto.WhiteNoiseIDfromString(idOri.String())
	if err != nil {
		t.Fatal(err)
	}
	privP2P := acc.GetP2PPrivKey()
	idP2P, err := peer.IDFromPublicKey(privP2P.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	idP2P2, err := id.GetPeerID()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, idP2P.String(), idP2P2.String())

	msg := []byte("hello white noise")

	cyphertext, err := acc.pubKey.ECIESEncrypt(msg, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := acc.privKey.ECIESDecrypt(cyphertext)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, msg, plaintext)
}