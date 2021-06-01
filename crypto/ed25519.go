package crypto

import (
	"crypto/sha512"
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/encrypt/ecies"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"golang.org/x/crypto/ed25519"
	"io"
)

type Ed25519PrivateKey struct {
	priv ed25519.PrivateKey
}

func (e Ed25519PrivateKey) Public() PublicKey {
	pub := e.priv.Public().(ed25519.PublicKey)
	return Ed25519PublicKey{pub: pub}
}

func (e Ed25519PrivateKey) Bytes() []byte {
	return MarshallEd25519PrivateKey(e)
}

func (e Ed25519PrivateKey) GetP2PKeypair() (crypto.PrivKey, crypto.PubKey, error) {
	pbmes := new(pb.PrivateKey)
	pbmes.Type = pb.KeyType_Ed25519
	buf := make([]byte, len(e.priv))
	copy(buf, e.priv)
	pbmes.Data = buf
	privBytes, err := proto.Marshal(pbmes)
	if err != nil {
		return nil, nil, err
	}

	p2pPriv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		return nil, nil, err
	}
	return p2pPriv, p2pPriv.GetPublic(), nil
}

func (e Ed25519PrivateKey) ECIESDecrypt(cyphertext []byte) ([]byte, error) {
	priv, _, err := ECIESKeypairFromEd25519(e.priv)
	if err != nil {
		return nil, err
	}
	suite := edwards25519.NewBlakeSHA256Ed25519()
	return ecies.Decrypt(suite, priv, cyphertext, suite.Hash)
}

type Ed25519PublicKey struct {
	pub ed25519.PublicKey
}

func (e Ed25519PublicKey) Bytes() []byte {
	return MarshallEd25519PublicKey(e)
}

func (e Ed25519PublicKey) ECIESEncrypt(plaintext []byte, rand io.Reader) ([]byte, error) {
	pk, err := ECIESPKFromEd25519(e.pub)
	if err != nil {
		return nil, err
	}
	suite := edwards25519.NewBlakeSHA256Ed25519()
	return ecies.Encrypt(suite, pk, plaintext, suite.Hash)
}

func (e Ed25519PublicKey) GetWhiteNoiseID() WhiteNoiseID {
	id := WhiteNoiseID{}
	id[0] = Ed25519
	copy(id[1:], e.pub)
	return id
}

func (e Ed25519PublicKey) PeerID() (peer.ID, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(e.pub)
	if err != nil {
		return "", err
	}
	return peer.IDFromPublicKey(pk)
}

func Ed25519PublicKeyFromP2P(key *crypto.Ed25519PublicKey) (Ed25519PublicKey, error) {
	buf, err := key.Raw()
	if err != nil {
		return Ed25519PublicKey{}, err
	}
	return Ed25519PublicKey{buf}, err
}

func GenerateEd25519KeyPair(r io.Reader) (Ed25519PrivateKey, Ed25519PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(r)
	if err != nil {
		return Ed25519PrivateKey{}, Ed25519PublicKey{}, err
	}
	return Ed25519PrivateKey{priv: priv}, Ed25519PublicKey{pub: pub}, nil
}

func ECIESKeypairFromEd25519(sk ed25519.PrivateKey) (kyber.Scalar, kyber.Point, error) {
	sk = sk[:32]
	suite := edwards25519.NewBlakeSHA256Ed25519()
	finial := sha512.Sum512(sk)
	secretBytes := finial[:32]

	secretBytes[0] &= 248
	secretBytes[31] &= 63
	secretBytes[31] |= 64
	secretBytes[0] &= 248
	secretBytes[31] &= 127
	secretBytes[31] |= 64

	private := suite.Scalar()
	err := private.UnmarshalBinary(secretBytes)
	if err != nil {
		return nil, nil, err
	}
	public := suite.Point().Mul(private, nil)
	return private, public, err
}

func ECIESPKFromEd25519(pk ed25519.PublicKey) (kyber.Point, error) {
	suite := edwards25519.NewBlakeSHA256Ed25519()
	pub := suite.Point()
	err := pub.UnmarshalBinary(pk)
	return pub, err
}

func MarshallEd25519PublicKey(pk Ed25519PublicKey) []byte {
	return pk.pub
}

func UnMarshallEd25519PublicKey(buf []byte) (Ed25519PublicKey, error) {
	if len(buf) != ed25519.PublicKeySize {
		return Ed25519PublicKey{}, errors.New("not right size")
	}
	return Ed25519PublicKey{pub: ed25519.PublicKey(buf)}, nil
}

func MarshallEd25519PrivateKey(sk Ed25519PrivateKey) []byte {
	buf := make([]byte, len(sk.priv))
	copy(buf, sk.priv)
	return buf
}

func UnMarshallEd25519PrivateKey(data []byte) (Ed25519PrivateKey, error) {
	if len(data) != ed25519.PrivateKeySize {
		return Ed25519PrivateKey{}, errors.New("not right size")
	}
	return Ed25519PrivateKey{priv: ed25519.PrivateKey(data)}, nil
}
