package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/libp2p/go-libp2p-core/crypto"
	nc "github.com/nspcc-dev/neofs-crypto"
	"io"
)

//todo:add more curve choices
//spec256r1 implementation
var ECDSACurve = elliptic.P256()

func GenerateECDSAKeyPair(r io.Reader) (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(ECDSACurve, r)
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

func MarshallECDSAPrivateKey(priv *ecdsa.PrivateKey) ([]byte, error) {
	return x509.MarshalECPrivateKey(priv)
}

func UnMarshallECDSAPrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	return x509.ParseECPrivateKey(data)
}

func MarshallECDSAPublicKey(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("pub nil")
	}
	return nc.MarshalPublicKey(pub), nil
}

func UnMarshallECDSAPublicKey(data []byte) (*ecdsa.PublicKey, error) {
	pk := nc.UnmarshalPublicKey(data)
	if pk == nil {
		return nil, errors.New("UnMarshallECDSAPublicKey err")
	}
	return pk, nil
}

func ECDSAPublicKeyFromP2PPK(p2pPK crypto.PubKey) (*ecdsa.PublicKey, error) {
	raw, err := p2pPK.Raw()
	if err != nil {
		return nil, err
	}
	pk, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return nil, err
	}
	ecdsaPK, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not ecdsa pk")
	}
	return ecdsaPK, nil
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
