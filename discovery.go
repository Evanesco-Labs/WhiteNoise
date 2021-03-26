package whitenoise

import (
	"context"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	"time"
	"whitenoise/log"
)

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

type Discovery struct {
	PeerMap  map[core.PeerID]core.PeerAddrInfo
	peerChan chan core.PeerAddrInfo
	event    chan core.PeerAddrInfo
}

func (self *Discovery) run() {
	for {
		peer := <-self.peerChan
		log.Infof("get peer:%v\n", peer.ID)
		_, ok := self.PeerMap[peer.ID]
		if !ok {
			self.event <- peer
		}
		self.PeerMap[peer.ID] = peer
	}
}

//interface to be called when new  peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

//Initialize the MDNS service
func initMDNS(ctx context.Context, peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	// An hour might be a long long period in practical applications. But this is fine for us
	ser, err := discovery.NewMdnsService(ctx, peerhost, time.Hour, rendezvous)
	if err != nil {
		panic(err)
	}

	//register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	ser.RegisterNotifee(n)
	return n.PeerChan
}
