package host

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/multiformats/go-multiaddr"
	"whitenoise/common/config"
	"whitenoise/common/log"
)

func NewHost(ctx context.Context, cfg *config.NetworkConfig, priv crypto.PrivKey) (host.Host, *kaddht.IpfsDHT, error) {
	transport, err := noise.New(priv)
	if err != nil {
		return nil, nil, err
	}
	bootpeers := make([]peer.AddrInfo, 0)
	if cfg.BootStrapPeers != "" {
		maddr, err := multiaddr.NewMultiaddr(cfg.BootStrapPeers)
		if err != nil {
			log.Errorf("Parse bootstrap node address err %v", err)
			return nil, nil, err
		}
		bootinfo, _ := peer.AddrInfoFromP2pAddr(maddr)
		bootpeers = append(bootpeers, *bootinfo)
	}

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		mode := kaddht.Mode(kaddht.ModeServer)
		if cfg.Mode == config.ClientMode {
			mode = kaddht.Mode(kaddht.ModeClient)
		}
		dht, err = kaddht.New(ctx, h, mode,
			kaddht.ProtocolPrefix("/whitenoise_dht"),
			kaddht.BootstrapPeers(bootpeers...),
		)
		return dht, err
	}

	var h host.Host
	if cfg.Mode == config.ClientMode {
		h, err := libp2p.New(
			ctx,
			libp2p.Security(noise.ID, transport),
			libp2p.Identity(priv),
		)
		if err != nil {
			return nil, nil, err
		}
		return h, nil, nil
	}

	if cfg.Mode == config.BootMode {
		sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", "0.0.0.0", cfg.ListenPort))
		h, err = libp2p.New(
			ctx,
			libp2p.ListenAddrs(sourceMultiAddr),
			libp2p.Security(noise.ID, transport),
			libp2p.Identity(priv),
			libp2p.Routing(newDHT),
			libp2p.EnableNATService(),
			libp2p.EnableAutoRelay(),
			libp2p.NATPortMap(),
		)
		if err != nil {
			return nil, nil, err
		}
		return h, dht, nil
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", "0.0.0.0", cfg.ListenPort))
	h, err = libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Security(noise.ID, transport),
		libp2p.Identity(priv),
		libp2p.Routing(newDHT),
		libp2p.EnableAutoRelay(),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, nil, err
	}

	return h, dht, nil
}
