package whitenoise

import (
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"io"
	"whitenoise/log"
	"whitenoise/pb"
)

const ACK_PROTOCOL string = "ack"

type AckManager struct {
	TaskMap map[string]Task
}

type Task struct {
	Id      string
	channel chan Result
}

type Result struct {
	ok   bool
	data []byte
}

func NewAckManager() *AckManager {
	return &AckManager{TaskMap: make(map[string]Task)}
}

func (manager *AckManager) AckStreamHandler(stream network.Stream) {
	defer stream.Close()
	str := NewStream(stream)
	lBytes := make([]byte, 4)
	_, err := io.ReadFull(str.RW, lBytes)
	if err != nil {
		return
	}

	l := Bytes2Int(lBytes)
	payloadBytes := make([]byte, l)
	_, err = io.ReadFull(str.RW, payloadBytes)
	if err != nil {
		log.Warn("payload not enough bytes")
		return
	}

	var ack = pb.Ack{}
	err = proto.Unmarshal(payloadBytes, &ack)
	if err != nil {
		return
	}

	task, ok := manager.TaskMap[ack.CommandId]
	if !ok {
		log.Warnf("No such task %v", task.Id)
		return
	}

	task.channel <- Result{
		ok:   ack.Result,
		data: ack.Data,
	}
}

func (manager *AckManager) AddTask(id string) chan Result {
	task := Task{
		Id:      id,
		channel: make(chan Result),
	}
	manager.TaskMap[id] = task
	return task.channel
}
