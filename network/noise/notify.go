package noise

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	"time"
	"whitenoise/common"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/protocol/proxy"
	"whitenoise/protocol/relay"
)

type NoiseNotifiee struct {
	host          host.Host
	actCtx        *actor.RootContext
	proxyPid      *actor.PID
	relayPid      *actor.PID
	whiteListMode bool
}

func (n NoiseNotifiee) Listen(network network.Network, multiaddr multiaddr.Multiaddr) {}

func (n NoiseNotifiee) ListenClose(network network.Network, multiaddr multiaddr.Multiaddr) {}

func (n NoiseNotifiee) Connected(network network.Network, conn network.Conn) {
	if n.whiteListMode {
		if _, ok := config.WhiteListPeers[conn.RemotePeer()]; !ok {
			log.Debug("block connection from", conn.RemotePeer())
			conn.Close()
			//todo: dos attack add into black list
		}
	}
	until, _ := time.Parse(time.RFC3339, common.NetTimeUntil)
	n.host.Peerstore().AddAddr(conn.RemotePeer(), conn.RemoteMultiaddr(), time.Until(until))
}

func (n NoiseNotifiee) Disconnected(network network.Network, conn network.Conn) {
	//proxy handle client disconnect
	n.actCtx.Request(n.proxyPid, proxy.ReqUnregister{PeerId: conn.RemotePeer()})
	n.host.Peerstore().ClearAddrs(conn.RemotePeer())
}

func (n NoiseNotifiee) OpenedStream(network network.Network, stream network.Stream) {}

func (n NoiseNotifiee) ClosedStream(network network.Network, stream network.Stream) {
	//handle relay protocol stream closed
	n.actCtx.Request(n.relayPid, relay.ReqHandleStreamClosed{StreamId: stream.ID()})
}
