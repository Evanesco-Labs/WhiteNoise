package ack

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
)

type ReqAck struct {
	Ack    *pb.Ack
	PeerId peer.ID
}

type ResError struct {
	Err error
}

type ReqAddTask struct {
	T Task
}

type ReqDeleteTask struct {
	Id string
}

func (manager *AckManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ReqAck:
		err := manager.SendAck(msg.Ack, msg.PeerId)
		if err != nil {
			log.Warn("send ack error", err)
		}
	case ReqAddTask:
		manager.AddTask(msg.T)
	case ReqDeleteTask:
		manager.DeletTask(msg.Id)
	default:
	}
}
