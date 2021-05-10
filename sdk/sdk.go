package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"github.com/asaskevich/EventBus"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/mr-tron/base58"
	"time"
	"whitenoise/common"
	"whitenoise/common/account"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/network"
)

var BootStrapPeers string = ""

const NewCircuitTimeout = 10 * time.Second
const GetCircuitTopic string = common.NewSecureConnAnswerTopic
const GenCircuitSuccessTopic string = common.NewSecureConnCallerTopic

type SecureConnection interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	LocalWhiteNoiseID() string
	RemoteWhiteNoiseID() string
}

type Client interface {
	GetMainNetPeers(cnt int) ([]peer.ID, error)
	Register(proxy core.PeerID) error
	Dial(remoteID string) (SecureConnection, string, error)
	GetCircuit(sessionID string) (SecureConnection, bool)
	SendMessage(data []byte, sessionID string) error
	DisconnectCircuit(sessionID string) error
	GetWhiteNoiseID() string
	UnRegister()
	EventBus() EventBus.Bus
}

type WhiteNoiseClient struct {
	node              *network.Node
	NewCircuitTimeout time.Duration
}

func NewClient(ctx context.Context) (*WhiteNoiseClient, error) {
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		BootStrapPeers:   BootStrapPeers,
		Mode:             config.ClientMode,
	}
	acc := account.GetAccount()
	node, err := network.NewNode(ctx, &cfg, *acc)
	if err != nil {
		return nil, err
	}
	node.Start(&cfg)
	return &WhiteNoiseClient{
		node,
		NewCircuitTimeout,
	}, nil
}

func NewOneTimeClient(ctx context.Context) (*WhiteNoiseClient, error) {
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		BootStrapPeers:   BootStrapPeers,
		Mode:             config.ClientMode,
	}
	acc := account.NewOneTimeAccount()
	node, err := network.NewNode(ctx, &cfg, *acc)
	if err != nil {
		return nil, err
	}
	node.Start(&cfg)
	return &WhiteNoiseClient{
		node,
		NewCircuitTimeout,
	}, nil
}

func (sdk *WhiteNoiseClient) GetMainNetPeers(cnt int) ([]peer.ID, error) {
	peerInfos, err := sdk.node.NoiseService.GetMainnetPeers(cnt)
	log.Debugf("%v", peerInfos)
	if err != nil {
		return nil, err
	}
	peers := make([]peer.ID, len(peerInfos))
	for i, _ := range peerInfos {
		peers[i] = peerInfos[i].ID
		sdk.node.NoiseService.Host().Peerstore().AddAddrs(peerInfos[i].ID, peerInfos[i].Addrs, peerstore.PermanentAddrTTL)
	}
	return peers, nil
}

func (sdk *WhiteNoiseClient) Register(proxy core.PeerID) error {
	return sdk.node.NoiseService.RegisterProxy(proxy)
}

func (sdk *WhiteNoiseClient) Dial(remoteID string) (SecureConnection, string, error) {
	sessionID := generateSessionID(remoteID, sdk.GetWhiteNoiseID())
	err := sdk.node.NoiseService.NewCircuit(remoteID, sessionID)
	if err != nil {
		return nil, "", err
	}
	timeout := time.After(sdk.NewCircuitTimeout)
	for {
		time.Sleep(time.Millisecond * 10)
		if conn, ok := sdk.GetCircuit(sessionID); ok {
			return conn, sessionID, nil
		}

		select {
		case <-timeout:
			return nil, "", errors.New("new circuit failed timeout")
		default:
			continue
		}
	}
}

func (sdk *WhiteNoiseClient) GetCircuit(sessionID string) (SecureConnection, bool) {
	conn, ok := sdk.node.NoiseService.Relay().GetSecureConn(sessionID)
	return conn, ok
}

func (sdk *WhiteNoiseClient) SendMessage(data []byte, sessionID string) error {
	conn, ok := sdk.GetCircuit(sessionID)
	if !ok {
		return errors.New("circuit not exist")
	}
	_, err := conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (sdk *WhiteNoiseClient) DisconnectCircuit(sessionID string) error {
	return sdk.node.NoiseService.Relay().CloseCircuit(sessionID)
}

func (sdk *WhiteNoiseClient) GetWhiteNoiseID() string {
	id := sdk.node.NoiseService.Account.GetWhiteNoiseID()
	return id.String()
}

func (sdk *WhiteNoiseClient) EventBus() EventBus.Bus {
	return sdk.node.NoiseService.EventBus()
}

func (sdk *WhiteNoiseClient) UnRegister() {
	sdk.node.NoiseService.UnRegister()
}

func generateSessionID(remoteID string, localID string) string {
	t := time.Now().UnixNano()
	tBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tBytes, uint64(t))
	bi := append([]byte(localID), []byte(remoteID)...)
	bi = append(bi, tBytes...)
	hash := sha256.Sum256(bi)
	return base58.Encode(hash[:])
}
