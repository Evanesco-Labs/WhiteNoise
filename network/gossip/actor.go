package gossip

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/Evanesco-Labs/WhiteNoise/internal/actorMsg"
)

func (service *DHTService) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case actorMsg.ReqGossipJoint:
		err := service.GossipJoint(msg.DesHash, msg.NegCypher)
		ctx.Respond(actorMsg.ResError{Err: err})
	case actorMsg.ReqDHTPeers:
		peerInfos := service.GetDHTPeers(msg.Max)
		ctx.Respond(actorMsg.ResDHTPeers{PeerInfos: peerInfos})
	default:
		//log.Debugf("Gossip actor cannot handle this request %v", msg)
	}
}
