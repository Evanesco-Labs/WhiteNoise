package command

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	core "github.com/libp2p/go-libp2p-core"
)

type ReqExpendSession struct {
	Relay     core.PeerID
	Joint     core.PeerID
	SessionId string
}

type ResError struct {
	Err error
}

func (manager *CmdManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ReqExpendSession:
		err := manager.ExpendSession(msg.Relay, msg.Joint, msg.SessionId)
		ctx.Respond(ResError{
			Err: err,
		})
	default:
		//log.Debugf("Command actor cannot handle request %v", msg)
	}
}
