package crypto

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
)

const (
	MarshallECDSAPublicKeyLength     = 33
	MarshallEd25519PublicKeyLength   = 32
	MarshallSecp256k1PublicKeyLength = 33
)

const (
	Ed25519 = iota
	Secpk1
	ECDSA
)

const DefaultKeyType = Ed25519

type PrivateKey interface {
	Public() PublicKey
	Bytes() []byte
	GetP2PKeypair() (crypto.PrivKey, crypto.PubKey, error)
	ECIESDecrypt(cyphertext []byte) ([]byte, error)
}

type PublicKey interface {
	Bytes() []byte
	ECIESEncrypt(plaintext []byte, rand io.Reader) ([]byte, error)
	GetWhiteNoiseID() WhiteNoiseID
	PeerID() (peer.ID, error)
}

func GenerateKeyPair(keyType int, rand io.Reader) (PrivateKey, PublicKey, error) {
	switch keyType {
	case ECDSA:
		return GenerateECDSAKeyPair(rand)
	case Ed25519:
		return GenerateEd25519KeyPair(rand)
	case Secpk1:
		return GenerateSecp256k1KeyPair(rand)
	default:
		return nil, nil, errors.New("key type not support")
	}
}
