package crypto

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"golang.org/x/crypto/ed25519"
	"io"
)

func GenerateEd25519KeyPair(r io.Reader) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(r)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

func P2PKeypairFromEd25519(priv ed25519.PrivateKey) (crypto.PrivKey, crypto.PubKey, error) {
	//marshall private key
	pbmes := new(pb.PrivateKey)
	pbmes.Type = pb.KeyType_Ed25519
	buf := make([]byte, len(priv))
	copy(buf, priv)
	pbmes.Data = buf
	privBytes, err := proto.Marshal(pbmes)
	fmt.Println(len(privBytes))
	if err != nil {
		return nil, nil, err
	}

	p2pPriv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return nil, nil, err
	}
	return p2pPriv, p2pPriv.GetPublic(), nil
}

func ECIESKeypairFromEd25519(priv ed25519.PrivateKey) () {

}