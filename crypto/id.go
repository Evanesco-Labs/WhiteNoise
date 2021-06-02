package crypto

import (
	"crypto/sha256"
	"errors"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	"whitenoise/common/log"
)

const WhiteNoiseIDLength = 34

type WhiteNoiseID [WhiteNoiseIDLength]byte

func (id WhiteNoiseID) String() string {
	return base58.Encode(id[:])
}

func WhiteNoiseIDfromString(s string) (WhiteNoiseID, error) {
	id := WhiteNoiseID{}
	raw, err := base58.Decode(s)
	if err != nil {
		return id, errors.New("WhiteNoiseIDfromString err: " + err.Error())
	}
	copy(id[:], raw)
	return id, nil
}

func (id WhiteNoiseID) Hash() string {
	hash := sha256.Sum256(id[:])
	return base58.Encode(hash[:])
}

func (id WhiteNoiseID) PublicKey() (PublicKey, error) {
	keyType := int(id[0])
	raw := id[1:]
	switch keyType {
	case Ed25519:
		return UnMarshallEd25519PublicKey(raw[:MarshallEd25519PublicKeyLength])
	case ECDSA:
		return UnMarshallECDSAPublicKey(raw[:MarshallECDSAPublicKeyLength])
	case Secpk1:
		return UnMarshallSecp256k1PublicKey(raw[:MarshallSecp256k1PublicKeyLength])
	default:
		return nil, errors.New("key type not support")
	}
}

func (id WhiteNoiseID) GetPeerID() (peer.ID, error) {
	pk, err := id.PublicKey()
	if err != nil {
		return "", err
	}
	return pk.PeerID()
}

func WhiteNoiseIDFromP2PPK(publicP2P crypto.PubKey) string {
	switch publicP2P.(type) {
	case *crypto.Ed25519PublicKey:
		pk, err := Ed25519PublicKeyFromP2P(publicP2P.(*crypto.Ed25519PublicKey))
		if err != nil {
			log.Error(err)
			return ""
		}
		return pk.GetWhiteNoiseID().String()
	case *crypto.ECDSAPublicKey:
		pk, err := ECDSAPublicKeyFromP2P(publicP2P.(*crypto.ECDSAPublicKey))
		if err != nil {
			log.Error(err)
			return ""
		}
		return pk.GetWhiteNoiseID().String()
	case *crypto.Secp256k1PublicKey:
		pk, err := SecpPublicKeyFromP2P(publicP2P.(*crypto.Secp256k1PublicKey))
		if err != nil {
			log.Error(err)
			return ""
		}
		return pk.GetWhiteNoiseID().String()
	default:
		log.Error(errors.New("unsupport key type"))
		return ""
	}
}
