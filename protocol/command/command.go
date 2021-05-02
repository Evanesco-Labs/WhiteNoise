package command

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/asaskevich/EventBus"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"time"
	"whitenoise/common"
	"whitenoise/common/log"
	"whitenoise/internal/pb"
	"whitenoise/network/session"
	"whitenoise/protocol/ack"
	"whitenoise/protocol/relay"
	"whitenoise/secure"
)

const CMD_PROTOCOL string = "/cmd"

type CmdManager struct {
	host                 core.Host
	context              context.Context
	actorCtx             *actor.RootContext
	relayPid             *actor.PID
	ackPid               *actor.PID
	cmdPid               *actor.PID
	ExpendSessionTimeout time.Duration
	eb                   EventBus.Bus
}

func NewCmdHandler(host core.Host, ctx context.Context, actCtx *actor.RootContext, eb EventBus.Bus) *CmdManager {
	return &CmdManager{
		host:                 host,
		context:              ctx,
		actorCtx:             actCtx,
		ExpendSessionTimeout: common.ExpendSessionTimeout,
		eb:                   eb,
	}
}

func (manager *CmdManager) Pid() *actor.PID {
	return manager.cmdPid
}

func (manager *CmdManager) SetPid(relayPid *actor.PID, ackPid *actor.PID) {
	manager.relayPid = relayPid
	manager.ackPid = ackPid
}

func (manager *CmdManager) Start() {
	props := actor.PropsFromProducer(func() actor.Actor {
		return manager
	})
	manager.cmdPid = manager.actorCtx.Spawn(props)
}

func (manager *CmdManager) CmdStreamHandler(stream network.Stream) {
	defer stream.Close()
	str := session.NewStream(stream, manager.context)
	payloadBytes, err := str.RW.ReadMsg()
	if err != nil {
		return
	}

	var command = pb.Command{}
	err = proto.Unmarshal(payloadBytes, &command)
	if err != nil {
		log.Error("unmarshal err", err)

	}
	switch command.Type {
	case pb.Cmdtype_SessionExPend:
		log.Info("Get session expend cmd")
		var cmd = pb.SessionExpend{}
		err = proto.Unmarshal(command.Data, &cmd)
		if err != nil {
			break
		}
		ackMsg := pb.Ack{
			CommandId: command.CommandId,
			Result:    false,
			Data:      []byte{},
		}

		fut := manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqGetSession{Id: cmd.SessionId}, common.RequestFutureDuration)
		res, err := fut.Result()
		if err != nil {
			return
		}

		sess := res.(relay.ResGetSession).Session
		ok := res.(relay.ResGetSession).Ok

		if !ok {
			log.Warnf("No such session: %v", cmd.SessionId)
			manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: cmd.SessionId})
			ackMsg.Data = []byte("No such session")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		if sess.IsReady() {
			log.Warnf("Session is ready, ack as both entry and relay %v", cmd.SessionId)
			ackMsg.Result = true
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			log.Debug("send circuit success signal")
			msg := relay.NewCircuitSuccess(cmd.SessionId)
			fut := manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqSendRelay{
				SessionId: cmd.SessionId,
				Data:      msg,
			}, common.RequestFutureDuration)
			res, err := fut.Result()
			if err != nil {
				log.Error("send success msg err", err)
			}
			resErr := res.(relay.ResError).Err
			if resErr != nil {
				log.Error("send success msg err", err)
			}
			break
		}
		id, err := peer.Decode(cmd.PeerId)
		if err != nil {
			log.Warnf("Decode peerid %v err: %v", cmd.PeerId, err)
			manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: cmd.SessionId})

			ackMsg.Data = []byte("Decode peerid error")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		fut = manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqNewSessiontoPeer{
			PeerID:    id,
			SessionID: cmd.SessionId,
			MyRole:    common.RelayRole,
			OtherRole: common.JointRole,
		}, common.RequestFutureDuration)
		res, err = fut.Result()
		if err != nil {
			break
		}
		resErr := res.(relay.ResError).Err

		if resErr != nil {
			log.Errorf("NewSessionToPeer err: %v", resErr)
			manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: cmd.SessionId})
			ackMsg.Data = []byte("NewSessionToPeer error")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		} else {
			ackMsg.Result = true
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}

	case pb.Cmdtype_Disconnect:
		log.Info("get Disconnect cmd")
	}

}

func (manager *CmdManager) ExpendSession(relay core.PeerID, joint core.PeerID, sessionId string) error {
	stream, err := manager.host.NewStream(manager.context, relay, protocol.ID(CMD_PROTOCOL))
	if err != nil {
		return err
	}
	s := session.NewStream(stream, manager.context)
	cmd := pb.SessionExpend{
		SessionId: sessionId,
		PeerId:    joint.String(),
	}
	cmdData, err := proto.Marshal(&cmd)
	if err != nil {
		return err
	}
	pl := pb.Command{
		CommandId: "",
		Type:      pb.Cmdtype_SessionExPend,
		From:      manager.host.ID().String(),
		Data:      cmdData,
	}
	payloadNoId, err := proto.Marshal(&pl)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payloadNoId)
	pl.CommandId = secure.EncodeMSGIDHash(hash[:])

	data, err := proto.Marshal(&pl)
	if err != nil {
		return err
	}

	err = s.RW.WriteMsg(data)
	if err != nil {
		return err
	}

	task := ack.Task{
		Id:      pl.CommandId,
		Channel: make(chan ack.Result),
	}

	manager.actorCtx.Request(manager.ackPid, ack.ReqAddTask{T: task})

	defer manager.actorCtx.Request(manager.ackPid, ack.ReqDeleteTask{Id: pl.CommandId})
	select {
	case <-time.After(manager.ExpendSessionTimeout):
		return errors.New("timeout")
	case result := <-task.Channel:
		if result.Ok {
			return nil
		} else {
			return errors.New("cmd rejected: " + string(result.Data))
		}
	}
}
