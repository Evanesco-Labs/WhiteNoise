package relay

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/network/session"
	core "github.com/libp2p/go-libp2p-core"
)

type ReqGetSession struct {
	Id string
}

type ResGetSession struct {
	Session session.Session
	Ok      bool
}

type ReqAddSessionID struct {
	Id      string
	Session session.Session
}

type ResError struct {
	Err error
}

type ReqNewSessiontoPeer struct {
	PeerID    core.PeerID
	SessionID string
	MyRole    common.SessionRole
	OtherRole common.SessionRole
}

type ReqCloseCircuit struct {
	SessionId string
}

type ReqSendRelay struct {
	SessionId string
	Data      []byte
}

type ReqHandleStreamClosed struct {
	StreamId string
}

func (manager *RelayMsgManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ReqGetSession:
		sess, ok := manager.GetSession(msg.Id)
		ctx.Respond(ResGetSession{
			Session: sess,
			Ok:      ok,
		})
	case ReqNewSessiontoPeer:
		err := manager.NewSessionToPeer(msg.PeerID, msg.SessionID, msg.MyRole, msg.OtherRole)
		ctx.Respond(ResError{Err: err})
	case ReqSendRelay:
		err := manager.SendRelay(msg.SessionId, msg.Data)
		ctx.Respond(ResError{Err: err})
	case ReqAddSessionID:
		manager.AddSessionId(msg.Id, msg.Session)
	case ReqCloseCircuit:
		err := manager.CloseCircuit(msg.SessionId)
		if err != nil {
			log.Warn("Close circuit err", err)
		}
	case ReqHandleStreamClosed:
		if info, ok := manager.GetStream(msg.StreamId); ok {
			if info.sessionID != "" {
				manager.CloseCircuit(info.sessionID)
			}
			manager.DeleteStream(msg.StreamId)
		}

	default:
	}
}
