package crypto

import (
	"crypto/sha256"
	"errors"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
)

const WhiteNoiseIDLength = 34

type WhiteNoiseID [WhiteNoiseIDLength]byte

func (id WhiteNoiseID) String() string {
	prefix := id[0]
	switch prefix {
	case Ed25519:
		pkBytes := id[1 : MarshallEd25519PublicKeyLength+1]
		return "0" + base58.Encode(pkBytes)
	case Secpk1:
		pkBytes := id[1 : MarshallSecp256k1PublicKeyLength+1]
		return "1" + base58.Encode(pkBytes)
	case ECDSA:
		pkBytes := id[1 : MarshallECDSAPublicKeyLength+1]
		return "2" + base58.Encode(pkBytes)
	default:
		log.Error("not support key type")
		return ""
	}
}

func WhiteNoiseIDfromString(s string) (WhiteNoiseID, error) {
	id := WhiteNoiseID{}
	switch s[0:1] {
	case "0":
		pkBytes, err := base58.Decode(s[1:])
		if err != nil {
			return [34]byte{}, err
		}
		id[0] = Ed25519
		copy(id[1:], pkBytes)
	case "1":
		pkBytes, err := base58.Decode(s[1:])
		if err != nil {
			return [34]byte{}, err
		}
		id[0] = Secpk1
		copy(id[1:], pkBytes)
	case "2":
		pkBytes, err := base58.Decode(s[1:])
		if err != nil {
			return [34]byte{}, err
		}
		id[0] = ECDSA
		copy(id[1:], pkBytes)
	default:
		return [34]byte{}, errors.New("not support key type")
	}
	return id, nil
}

func (id WhiteNoiseID) Hash() string {
	hash := [32]byte{}
	switch id[0] {
	case Ed25519:
		hash = sha256.Sum256(id[1 : MarshallEd25519PublicKeyLength+1])
	case Secpk1:
		hash = sha256.Sum256(id[1 : MarshallSecp256k1PublicKeyLength+1])
	case ECDSA:
		hash = sha256.Sum256(id[1 : MarshallECDSAPublicKeyLength+1])
	}
	//hash = sha256.Sum256(id[:])
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
