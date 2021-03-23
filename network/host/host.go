package host

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/multiformats/go-multiaddr"
	"whitenoise/common/config"
)

func NewHost(ctx context.Context, cfg *config.NetworkConfig,priv crypto.PrivKey) (host.Host, *kaddht.IpfsDHT, error) {
	//priv := crypto.GenPrivKey()
	//if nil == priv {
	//	return nil, nil, errors.New("gen PrivKey err in NewDummyHost")
	//}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.ListenHost, cfg.ListenPort))
	transport, err := noise.New(priv)
	if err != nil {
		return nil, nil, err
	}

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		mode := kaddht.Mode(kaddht.ModeServer)
		if cfg.Mode == config.ClientMode {
			mode = kaddht.Mode(kaddht.ModeClient)
		}
		dht, err = kaddht.New(ctx, h, mode)
		return dht, err
	}

	h, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Security(noise.ID, transport),
		libp2p.Identity(priv),
		libp2p.Routing(newDHT),
	)
	if err != nil {
		return nil, nil, err
	}

	return h, dht, nil
}
