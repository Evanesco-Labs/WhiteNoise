package secure

import (
	"context"
	"github.com/flynn/noise"
	"sync"
	"time"
	"whitenoise/common"
	"whitenoise/common/account"
	"whitenoise/protocol/relay"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type SecureSession struct {
	initiator bool

	localID   peer.ID
	localKey  crypto.PrivKey
	remoteID  peer.ID
	remoteKey crypto.PubKey

	readLock  sync.Mutex
	writeLock sync.Mutex
	insecure  relay.InsecureConn

	qseek int     // queued bytes seek value.
	qbuf  []byte  // queued bytes buffer.
	rlen  [2]byte // work buffer to read in the incoming message length.

	enc *noise.CipherState
	dec *noise.CipherState

	readHandshakeMsgTimeout time.Duration
}

func NewSecureSession(localID peer.ID, privateKey crypto.PrivKey, ctx context.Context, insecure relay.InsecureConn, remote peer.ID, initiator bool) (*SecureSession, error) {
	s := &SecureSession{
		insecure:                insecure,
		initiator:               initiator,
		localID:                 localID,
		localKey:                privateKey,
		remoteID:                remote,
		readHandshakeMsgTimeout: common.ReadHandShakeMsgTimeout,
	}

	respCh := make(chan error, 1)
	go func() {

		respCh <- s.runHandshake(ctx)
	}()

	select {
	case err := <-respCh:
		if err != nil {
			_ = s.insecure.Close()
		}
		return s, err

	case <-ctx.Done():
		_ = s.insecure.Close()
		<-respCh
		return nil, ctx.Err()
	}
}

func (s *SecureSession) LocalPeer() peer.ID {
	return s.localID
}

func (s *SecureSession) LocalPrivateKey() crypto.PrivKey {
	return s.localKey
}

func (s *SecureSession) LocalPublicKey() crypto.PubKey {
	return s.localKey.GetPublic()
}

func (s *SecureSession) LocalWhiteNoiseID() string {
	whitenoiseID, err := account.WhitenoiseIDFromP2PPK(s.localKey.GetPublic())
	if err != nil {
		return ""
	}
	return whitenoiseID.String()
}

func (s *SecureSession) RemotePeer() peer.ID {
	return s.remoteID
}

func (s *SecureSession) RemotePublicKey() crypto.PubKey {
	return s.remoteKey
}

func (s *SecureSession) RemoteWhiteNoiseID() string {
	whitenoiseID, err := account.WhitenoiseIDFromP2PPK(s.remoteKey)
	if err != nil {
		return ""
	}
	return whitenoiseID.String()
}

func (s *SecureSession) Close() error {
	return s.insecure.Close()
}
