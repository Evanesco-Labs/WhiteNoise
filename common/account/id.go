package account

import (
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	wnct "whitenoise/crypto"
)

type WhiteNoiseID []byte

func (id WhiteNoiseID) String() string {
	return base58.Encode(id)
}

func WhiteNoiseIDfromString(s string) (WhiteNoiseID, error) {
	raw, err := base58.Decode(s)
	if err != nil {
		return nil, errors.New("WhiteNoiseIDfromString err: " + err.Error())
	}
	return WhiteNoiseID(raw), nil
}

func (id WhiteNoiseID) Hash() string {
	hash := sha256.Sum256(id)
	return base58.Encode(hash[:])
}

//todo:We must marshal ecdsa pk the same way as libp2p to get PeerId from WhiteNoiseId. Fix this.
func PeerIDFromWhiteNoiseID(id WhiteNoiseID) (peer.ID, error) {
	ecdsaPk, err := wnct.UnMarshallECDSAPublicKey([]byte(id))
	if err != nil {
		return "", err
	}
	p2pPKString, err := x509.MarshalPKIXPublicKey(ecdsaPk)
	if err != nil {
		return "", err
	}
	ecdsaPK, err := crypto.UnmarshalECDSAPublicKey(p2pPKString)
	if err != nil {
		return "", err
	}
	return peer.IDFromPublicKey(ecdsaPK)
}

func ECIESPKFromWhiteNoiseID(id WhiteNoiseID) (*ecies.PublicKey, error) {
	ecdsaPK, err := wnct.UnMarshallECDSAPublicKey([]byte(id))
	if err != nil {
		return nil, err
	}
	return wnct.ECIESPKFromECDSA(ecdsaPK), nil
}

func WhitenoiseIDFromP2PPK(pk crypto.PubKey) (WhiteNoiseID, error) {
	ecdsaPK, err := wnct.ECDSAPublicKeyFromP2PPK(pk)
	if err != nil {
		return nil, err
	}
	whitenoiseID, err := wnct.MarshallECDSAPublicKey(ecdsaPK)
	if err != nil {
		return nil, err
	}
	return WhiteNoiseID(whitenoiseID), nil
}
