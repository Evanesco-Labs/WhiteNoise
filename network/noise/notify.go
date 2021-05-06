package noise

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	"whitenoise/protocol/proxy"
	"whitenoise/protocol/relay"
)

type NoiseNotifiee struct {
	actCtx   *actor.RootContext
	proxyPid *actor.PID
	relayPid *actor.PID
}

func (n NoiseNotifiee) Listen(network network.Network, multiaddr multiaddr.Multiaddr) {}

func (n NoiseNotifiee) ListenClose(network network.Network, multiaddr multiaddr.Multiaddr) {}

func (n NoiseNotifiee) Connected(network network.Network, conn network.Conn) {}

func (n NoiseNotifiee) Disconnected(network network.Network, conn network.Conn) {
	//proxy handle client disconnect
	n.actCtx.Request(n.proxyPid, proxy.ReqUnregister{PeerId: conn.RemotePeer()})
}

func (n NoiseNotifiee) OpenedStream(network network.Network, stream network.Stream) {}

func (n NoiseNotifiee) ClosedStream(network network.Network, stream network.Stream) {
	//handle relay protocol stream closed
	n.actCtx.Request(n.relayPid, relay.ReqHandleStreamClosed{StreamId: stream.ID()})
}
