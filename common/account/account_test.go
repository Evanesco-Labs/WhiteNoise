package account

import (
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"testing"
	"whitenoise/crypto"
)

func TestKeyID(t *testing.T) {
	acc := GetAccount()
	idOri := acc.GetWhiteNoiseID()
	id, err := WhiteNoiseIDfromString(idOri.String())
	fmt.Println("whitenoiseID: ", id.String())
	privP2P := acc.GetP2PPrivKey()
	privECIES := acc.GetECIESPrivKey()
	idP2P, err := peer.IDFromPublicKey(privP2P.GetPublic())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PeerID1: ", idP2P.String())
	idP2P2, err := PeerIDFromWhiteNoiseID(id)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PeerID2: ", idP2P2.String())

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
	fmt.Println(string(plaintext))
}
