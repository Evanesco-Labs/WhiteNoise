package gossip

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"whitenoise/common/log"
	"whitenoise/internal/actorMsg"
)

func (service *DHTService) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case actorMsg.ReqGossipJoint:
		log.Debugf("receive request %v", msg)
		err := service.GossipJoint(msg.DesHash, msg.Joint, msg.SessionId)
		ctx.Respond(actorMsg.ResError{Err: err})
	default:
		//log.Debugf("Gossip actor cannot handle this request %v", msg)
	}
}
