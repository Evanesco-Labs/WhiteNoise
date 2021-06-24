package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	nc "github.com/nspcc-dev/neofs-crypto"
	"io"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
)

//todo:add more curve choices
//spec256r1 implementation
var ECDSACurve = elliptic.P256()

type ECDSAPrivateKey struct {
	Priv *ecdsa.PrivateKey
}

func (E ECDSAPrivateKey) Public() PublicKey {
	return ECDSAPublicKey{Pub: &E.Priv.PublicKey}
}

func (E ECDSAPrivateKey) Bytes() []byte {
	b, err := x509.MarshalECPrivateKey(E.Priv)
	if err != nil {
		log.Error("Marshall ecdsa private key error", err)
		return nil
	}
	return b
}

func (E ECDSAPrivateKey) GetP2PKeypair() (crypto.PrivKey, crypto.PubKey, error) {
	return crypto.ECDSAKeyPairFromKey(E.Priv)
}

func (E ECDSAPrivateKey) ECIESDecrypt(cyphertext []byte) ([]byte, error) {
	eciesPriv := ECIESKeypairFromECDSA(E.Priv)
	return eciesPriv.Decrypt(cyphertext, nil, nil)
}

type ECDSAPublicKey struct {
	Pub *ecdsa.PublicKey
}

func (E ECDSAPublicKey) Bytes() []byte {
	b, err := MarshallECDSAPublicKey(E)
	if err != nil {
		log.Error("Marshall ECDSA public key error", err)
		return nil
	}
	return b
}

func (E ECDSAPublicKey) ECIESEncrypt(plaintext []byte, rand io.Reader) ([]byte, error) {
	eciesPub := ECIESPKFromECDSA(E.Pub)
	return ecies.Encrypt(rand, eciesPub, plaintext, nil, nil)
}

func (E ECDSAPublicKey) GetWhiteNoiseID() WhiteNoiseID {
	id := WhiteNoiseID{}
	id[0] = byte(ECDSA)
	b, err := MarshallECDSAPublicKey(E)
	if err != nil {
		log.Error(err)
		return id
	}
	copy(id[1:], b)
	return id
}

func (E ECDSAPublicKey) PeerID() (peer.ID, error) {
	p2pPKString, err := x509.MarshalPKIXPublicKey(E.Pub)
	if err != nil {
		return "", err
	}
	ecdsaPK, err := crypto.UnmarshalECDSAPublicKey(p2pPKString)
	if err != nil {
		return "", err
	}
	return peer.IDFromPublicKey(ecdsaPK)
}

func GenerateECDSAKeyPair(r io.Reader) (PrivateKey, PublicKey, error) {
	ecdsaPriv, err := ecdsa.GenerateKey(ECDSACurve, r)
	if err != nil {
		return nil, nil, err
	}
	return ECDSAPrivateKey{Priv: ecdsaPriv}, ECDSAPublicKey{Pub: &ecdsaPriv.PublicKey}, err
}

func P2PKeypairFromECDSA(priv *ecdsa.PrivateKey) (crypto.PrivKey, crypto.PubKey, error) {
	return crypto.ECDSAKeyPairFromKey(priv)
}

func ECIESKeypairFromECDSA(priv *ecdsa.PrivateKey) *ecies.PrivateKey {
	return ecies.ImportECDSA(priv)
}

func ECIESPKFromECDSA(pub *ecdsa.PublicKey) *ecies.PublicKey {
	return ecies.ImportECDSAPublic(pub)
}

func MarshallECDSAPrivateKey(priv ECDSAPrivateKey) ([]byte, error) {
	return x509.MarshalECPrivateKey(priv.Priv)
}

func UnMarshallECDSAPrivateKey(data []byte) (ECDSAPrivateKey, error) {
	priv, err := x509.ParseECPrivateKey(data)
	if err != nil {
		return ECDSAPrivateKey{}, err
	}
	return ECDSAPrivateKey{Priv: priv}, err
}

// MarshallECDSAPublicKey todo::change marshall scheme
func MarshallECDSAPublicKey(pub ECDSAPublicKey) ([]byte, error) {
	if pub.Pub == nil {
		return nil, errors.New("pub nil")
	}
	return nc.MarshalPublicKey(pub.Pub), nil
}

func UnMarshallECDSAPublicKey(data []byte) (ECDSAPublicKey, error) {
	pk := nc.UnmarshalPublicKey(data)
	if pk == nil {
		return ECDSAPublicKey{}, errors.New("UnMarshallECDSAPublicKey err")
	}
	return ECDSAPublicKey{Pub: pk}, nil
}

func ECDSAPublicKeyFromP2P(p2pPK *crypto.ECDSAPublicKey) (ECDSAPublicKey, error) {
	raw, err := p2pPK.Raw()
	if err != nil {
		return ECDSAPublicKey{}, err
	}
	pk, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return ECDSAPublicKey{}, err
	}
	ecdsaPK, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		return ECDSAPublicKey{}, errors.New("not ecdsa pk")
	}
	return ECDSAPublicKey{ecdsaPK}, nil
}

func ECIESEncrypt(key *ecies.PublicKey, plaintext []byte, r io.Reader) ([]byte, error) {
	return ecies.Encrypt(r, key, plaintext, nil, nil)
}

func ECIESDecrypt(key *ecies.PrivateKey, cyphertext []byte) ([]byte, error) {
	return key.Decrypt(cyphertext, nil, nil)
}

func EncodeEcdsaPriv(privateKey *ecdsa.PrivateKey) (string, error) {
	x509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	return string(pemEncoded), nil
}

func DecodeEcdsaPriv(pemEncoded string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}
