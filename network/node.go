package network

import (
	"context"
	"errors"
	"github.com/AsynkronIT/protoactor-go/actor"
	core "github.com/libp2p/go-libp2p-core"
	"whitenoise/common/account"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/network/gossip"
	"whitenoise/network/host"
	"whitenoise/network/noise"
)

type Node struct {
	NoiseService *noise.NoiseService
	DHTService   *gossip.DHTService
}

func NewNode(ctx context.Context, cfg *config.NetworkConfig, acc account.Account) (*Node, error) {
	whiteNoiseID := acc.GetWhiteNoiseID()
	log.Info("WhiteNoiseID:", whiteNoiseID.String())
	priv := acc.GetP2PPrivKey()
	if nil == priv {
		return nil, errors.New("gen PrivKey err in NewDummyHost")
	}

	h, dht, err := host.NewHost(ctx, cfg, priv)
	if err != nil {
		return nil, err
	}
	system := actor.NewActorSystem()
	noiseService, err := noise.NewNoiseService(ctx, system.Root, cfg, h, priv, acc)
	if err != nil {
		return nil, err
	}
	if cfg.Mode == config.ClientMode {
		return &Node{
			NoiseService: noiseService,
		}, nil
	}
	pubsubService, err := gossip.NewDHTService(ctx, system.Root, cfg, h, dht)
	if err != nil {
		return nil, err
	}
	return &Node{
		NoiseService: noiseService,
		DHTService:   pubsubService,
	}, nil
}

func (node *Node) Start(cfg *config.NetworkConfig) {
	if cfg.Mode == config.ClientMode {
		node.NoiseService.Start()
		node.NoiseService.SetPid(nil)
		node.NoiseService.SetNotify(node.Host(),cfg)
		return
	}
	node.DHTService.Start(cfg)
	node.NoiseService.Start()
	node.NoiseService.SetPid(node.DHTService.Pid())
	node.DHTService.SetPid(node.NoiseService.ProxyPid(), node.NoiseService.RelayPid(), node.NoiseService.CmdPid())
	node.NoiseService.SetNotify(node.Host(),cfg)
}

func (node *Node) Host() core.Host {
	return node.NoiseService.Host()
}
