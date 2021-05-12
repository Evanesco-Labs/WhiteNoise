package ack

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/asaskevich/EventBus"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"sync"
	"whitenoise/common/log"
	"whitenoise/internal/pb"
	"whitenoise/network/session"
)

const ACK_PROTOCOL string = "/ack"

type AckManager struct {
	context  context.Context
	actorCtx *actor.RootContext
	TaskMap  sync.Map
	ackPid   *actor.PID
	host     core.Host
	eb       EventBus.Bus
}

type Task struct {
	Id      string
	Channel chan Result
}

type Result struct {
	Ok   bool
	Data []byte
}

func NewAckManager(host core.Host, ctx context.Context, actCtx *actor.RootContext, eb EventBus.Bus) *AckManager {
	return &AckManager{
		context:  ctx,
		actorCtx: actCtx,
		TaskMap:  sync.Map{},
		host:     host,
		eb:       eb,
	}
}

func (manager *AckManager) Start() {
	props := actor.PropsFromProducer(func() actor.Actor {
		return manager
	})
	manager.ackPid = manager.actorCtx.Spawn(props)
}

func (manager *AckManager) Pid() *actor.PID {
	return manager.ackPid
}

func (manager *AckManager) AckStreamHandler(stream network.Stream) {
	defer stream.Close()
	str := session.NewStream(stream, manager.context)
	payloadBytes, err := str.RW.ReadMsg()
	if err != nil {
		return
	}

	var ack = pb.Ack{}
	err = proto.Unmarshal(payloadBytes, &ack)
	if err != nil {
		return
	}

	res, ok := manager.TaskMap.Load(ack.CommandId)
	task := res.(Task)
	if !ok {
		log.Warnf("No such task %v", task.Id)
		return
	}

	task.Channel <- Result{
		Ok:   ack.Result,
		Data: ack.Data,
	}
}

func (manager *AckManager) AddTask(task Task) {
	manager.TaskMap.Store(task.Id, task)
}

func (manager *AckManager) SendAck(ack *pb.Ack, peerId core.PeerID) error {
	stream, err := manager.host.NewStream(manager.context, peerId, core.ProtocolID(ACK_PROTOCOL))
	defer stream.Close()
	if err != nil {
		return err
	}
	data, err := proto.Marshal(ack)
	if err != nil {
		return err
	}

	s := session.NewStream(stream, manager.context)

	err = s.RW.WriteMsg(data)
	if err != nil {
		return err
	}
	return nil
}

func (manager *AckManager) DeletTask(id string) {
	if v, ok := manager.TaskMap.Load(id); ok {
		close(v.(Task).Channel)
	}
	manager.TaskMap.Delete(id)
}
