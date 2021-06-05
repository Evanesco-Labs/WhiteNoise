package secure

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/flynn/noise"
	"github.com/golang/protobuf/proto"
	pool "github.com/libp2p/go-buffer-pool"
	"golang.org/x/crypto/poly1305"
	"whitenoise/common/log"
	"whitenoise/internal/pb"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

const payloadSigPrefix = "noise-libp2p-static-key:"

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

func (s *SecureSession) runHandshake(ctx context.Context) error {
	log.Debug("run handshake")
	kp, err := noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return fmt.Errorf("error generating static keypair: %w", err)
	}
	cfg := noise.Config{
		CipherSuite:   cipherSuite,
		Pattern:       noise.HandshakeXX,
		Initiator:     s.initiator,
		StaticKeypair: kp,
	}

	hs, err := noise.NewHandshakeState(cfg)
	if err != nil {
		return fmt.Errorf("error initializing handshake state: %w", err)
	}
	payload, err := s.generateHandshakePayload(kp)
	if err != nil {
		return err
	}

	maxMsgSize := 2*noise.DH25519.DHLen() + len(payload) + 2*poly1305.TagSize
	hbuf := pool.Get(maxMsgSize + LengthPrefixLength)
	defer pool.Put(hbuf)
	if s.initiator {
		log.Debug("caller stage 0")
		err = s.sendHandshakeMessage(hs, nil, hbuf)
		if err != nil {
			return fmt.Errorf("error sending handshake message: %w", err)
		}

		log.Debug("caller stage 1")
		plaintext, err := s.readHandshakeMessage(hs)
		if err != nil {
			return fmt.Errorf("error reading handshake message: %w", err)
		}
		err = s.handleRemoteHandshakePayload(plaintext, hs.PeerStatic())
		if err != nil {
			return err
		}

		log.Debug("caller stage 2")
		err = s.sendHandshakeMessage(hs, payload, hbuf)
		if err != nil {
			return fmt.Errorf("error sending handshake message: %w", err)
		}
	} else {
		log.Debug("answer stage 0")
		plaintext, err := s.readHandshakeMessage(hs)
		if err != nil {
			return fmt.Errorf("error reading handshake message: %w", err)
		}

		log.Debug("answer stage 1")
		err = s.sendHandshakeMessage(hs, payload, hbuf)
		if err != nil {
			return fmt.Errorf("error sending handshake message: %w", err)
		}

		log.Debug("answer stage 2")
		plaintext, err = s.readHandshakeMessage(hs)
		if err != nil {
			return fmt.Errorf("error reading handshake message: %w", err)
		}
		err = s.handleRemoteHandshakePayload(plaintext, hs.PeerStatic())
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SecureSession) setCipherStates(cs1, cs2 *noise.CipherState) {
	if s.initiator {
		s.enc = cs1
		s.dec = cs2
	} else {
		s.enc = cs2
		s.dec = cs1
	}
}

func (s *SecureSession) sendHandshakeMessage(hs *noise.HandshakeState, payload []byte, hbuf []byte) error {
	bz, cs1, cs2, err := hs.WriteMessage(hbuf[:LengthPrefixLength], payload)
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint16(bz, uint16(len(bz)-LengthPrefixLength))

	_, err = s.writeMsgInsecure(bz)
	if err != nil {
		return err
	}

	if cs1 != nil && cs2 != nil {
		s.setCipherStates(cs1, cs2)
	}
	return nil
}

func (s *SecureSession) readHandshakeMessage(hs *noise.HandshakeState) ([]byte, error) {
	l, err := s.readNextInsecureMsgLen()
	if err != nil {
		return nil, err
	}

	buf := pool.Get(l)
	defer pool.Put(buf)
	if err := s.readNextMsgInsecure(buf); err != nil {
		return nil, err
	}
	msg, cs1, cs2, err := hs.ReadMessage(nil, buf)
	if err != nil {
		return nil, err
	}
	if cs1 != nil && cs2 != nil {
		s.setCipherStates(cs1, cs2)
	}
	return msg, nil
}

func (s *SecureSession) generateHandshakePayload(localStatic noise.DHKey) ([]byte, error) {
	localKeyRaw, err := s.LocalPublicKey().Bytes()
	if err != nil {
		return nil, fmt.Errorf("error serializing libp2p identity key: %w", err)
	}

	toSign := append([]byte(payloadSigPrefix), localStatic.Public...)
	signedPayload, err := s.localKey.Sign(toSign)
	if err != nil {
		return nil, fmt.Errorf("error sigining handshake payload: %w", err)
	}
	payload := new(pb.NoiseHandshakePayload)
	payload.IdentityKey = localKeyRaw
	payload.IdentitySig = signedPayload
	payloadEnc, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling handshake payload: %w", err)
	}
	return payloadEnc, nil
}

func (s *SecureSession) handleRemoteHandshakePayload(payload []byte, remoteStatic []byte) error {
	nhp := new(pb.NoiseHandshakePayload)
	err := proto.Unmarshal(payload, nhp)
	if err != nil {
		return fmt.Errorf("error unmarshaling remote handshake payload: %w", err)
	}

	remotePubKey, err := crypto.UnmarshalPublicKey(nhp.GetIdentityKey())
	if err != nil {
		return err
	}
	id, err := peer.IDFromPublicKey(remotePubKey)
	if err != nil {
		return err
	}

	if s.initiator && s.remoteID != id {
		return fmt.Errorf("peer id mismatch: expected %s, but remote key matches %s", s.remoteID.Pretty(), id.Pretty())
	}

	sig := nhp.GetIdentitySig()
	msg := append([]byte(payloadSigPrefix), remoteStatic...)
	ok, err := remotePubKey.Verify(msg, sig)
	if err != nil {
		return fmt.Errorf("error verifying signature: %w", err)
	} else if !ok {
		return fmt.Errorf("handshake signature invalid")
	}

	s.remoteID = id
	s.remoteKey = remotePubKey
	return nil
}
