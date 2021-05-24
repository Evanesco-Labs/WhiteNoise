package account

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/magiconair/properties/assert"
	"os"
	"testing"
	"whitenoise/crypto"
)

func TestKeyID(t *testing.T) {
	acc := GetAccount()
	idOri := acc.GetWhiteNoiseID()
	id, err := WhiteNoiseIDfromString(idOri.String())
	privP2P := acc.GetP2PPrivKey()
	privECIES := acc.GetECIESPrivKey()
	idP2P, err := peer.IDFromPublicKey(privP2P.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	idP2P2, err := PeerIDFromWhiteNoiseID(id)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, idP2P.String(), idP2P2.String())
	pkECIES, err := ECIESPKFromWhiteNoiseID(id)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("hello white noise")

	cyphertext, err := crypto.ECIESEncrypt(pkECIES, msg, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	plaintext, err := crypto.ECIESDecrypt(privECIES, cyphertext)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, msg, plaintext)
}

func Test_GetAccountFromFile(t *testing.T) {
	r := rand.Reader
	priv, err := crypto.GenerateECDSAKeyPair(r)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes, err := crypto.MarshallECDSAPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	privBytes, err := crypto.MarshallECDSAPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := crypto.EncodeEcdsaPriv(priv)
	if err != nil {
		t.Fatal(err)
	}

	path := "testaccount.pem"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(encoded)
	if err != nil {
		t.Fatal(err)
	}

	acc := GetAccountFromFile(path)

	assert.Equal(t, acc.privKey, privBytes)
	assert.Equal(t, acc.pubKey, pubBytes)
}
