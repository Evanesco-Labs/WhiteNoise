package crypto

import (
	"crypto/ecdsa"
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/Evanesco-Labs/go-evanesco/crypto/ecies"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
)

type Secp256k1PublicKey struct {
	Pub *ecdsa.PublicKey
}

func (s Secp256k1PublicKey) Bytes() []byte {
	return MarshallSecp256k1PublicKey(s)
}

func (s Secp256k1PublicKey) ECIESEncrypt(plaintext []byte, rand io.Reader) ([]byte, error) {
	eciesPK := ecies.ImportECDSAPublic(s.Pub)
	ecies.AddParamsForCurve(btcec.S256(), ecies.ECIES_AES128_SHA256)
	return ecies.Encrypt(rand, eciesPK, plaintext, nil, nil)
}

func (s Secp256k1PublicKey) GetWhiteNoiseID() WhiteNoiseID {
	id := WhiteNoiseID{}
	id[0] = byte(Secpk1)
	b := MarshallSecp256k1PublicKey(s)
	copy(id[1:], b)
	return id
}

func (s Secp256k1PublicKey) PeerID() (peer.ID, error) {
	p2pPk := (*crypto.Secp256k1PublicKey)((*btcec.PublicKey)(s.Pub))
	return peer.IDFromPublicKey(p2pPk)
}

type Secp256k1PrivateKey struct {
	Priv *ecdsa.PrivateKey
}

func (s Secp256k1PrivateKey) Public() PublicKey {
	return Secp256k1PublicKey{&s.Priv.PublicKey}
}

func (s Secp256k1PrivateKey) Bytes() []byte {
	return MarshallSecp256k1PrivateKey(s)
}

func (s Secp256k1PrivateKey) GetP2PKeypair() (crypto.PrivKey, crypto.PubKey, error) {
	privk := (*crypto.Secp256k1PrivateKey)(s.Priv)
	return privk, privk.GetPublic(), nil
}

func (s Secp256k1PrivateKey) ECIESDecrypt(cyphertext []byte) ([]byte, error) {
	key := ecies.ImportECDSA(s.Priv)
	ecies.AddParamsForCurve(btcec.S256(), ecies.ECIES_AES128_SHA256)
	return key.Decrypt(cyphertext, nil, nil)
}

func GenerateSecp256k1KeyPair(r io.Reader) (PrivateKey, PublicKey, error) {
	privk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}
	privECDSA := privk.ToECDSA()
	return Secp256k1PrivateKey{Priv: privECDSA}, Secp256k1PublicKey{Pub: &privECDSA.PublicKey}, err
}

func MarshallSecp256k1PublicKey(pk Secp256k1PublicKey) []byte {
	return (*btcec.PublicKey)(pk.Pub).SerializeCompressed()
}

func UnMarshallSecp256k1PublicKey(data []byte) (Secp256k1PublicKey, error) {
	pk, err := btcec.ParsePubKey(data, btcec.S256())
	if err != nil {
		return Secp256k1PublicKey{}, err
	}
	return Secp256k1PublicKey{pk.ToECDSA()}, nil
}

func MarshallSecp256k1PrivateKey(sk Secp256k1PrivateKey) []byte {
	privk := btcec.PrivateKey(*sk.Priv)
	return privk.Serialize()
}

func UnMarshallSecp256k1PrivateKey(data []byte) (Secp256k1PrivateKey, error) {
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), data)
	if priv == nil {
		return Secp256k1PrivateKey{}, errors.New("UnMarshallSecp256k1PrivateKey PrivKeyFromBytes error")
	}
	return Secp256k1PrivateKey{Priv: priv.ToECDSA()}, nil
}

func SecpPublicKeyFromP2P(p2pPK crypto.PubKey) (Secp256k1PublicKey, error) {
	raw, err := p2pPK.Raw()
	if err != nil {
		return Secp256k1PublicKey{}, err
	}
	return UnMarshallSecp256k1PublicKey(raw)
}
