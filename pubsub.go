package whitenoise

import (
	"context"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"whitenoise/log"
	"whitenoise/pb"
)

const GOSSIPTOPICNAME string = "gossip_topic"

type PubsubService struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan GossipMsg
	ctx      context.Context
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	network  *NetworkService
}

type GossipMsg struct {
	data []byte
}

func (service *NetworkService) NewPubsubService() error {
	ctx, _ := context.WithCancel(service.ctx)
	ps, err := pubsub.NewGossipSub(ctx, service.host, pubsub.WithNoAuthor()) //pubsub omit from,dig and change default id
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return err
	}
	topic, err := ps.Join(GOSSIPTOPICNAME)
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return err
	}

	pubsubService := PubsubService{
		Messages: make(chan GossipMsg),
		ctx:      ctx,
		ps:       ps,
		topic:    topic,
		sub:      sub,
		network:  service,
	}
	service.PubsubService = &pubsubService
	return nil
}

func (service *PubsubService) Start() {
	go service.handleMessage()
}

func (service *PubsubService) Subscribe() {

}

func (service *PubsubService) Publish(data []byte) error {
	return service.topic.Publish(service.ctx, data)
}

func (service *PubsubService) handleMessage() {
	for {
		msg, err := service.sub.Next(service.ctx)
		if err != nil {
			close(service.Messages)
			return
		}

		gossipMsg := GossipMsg{data: msg.Data}
		service.Messages <- gossipMsg
	}
}

func (service *PubsubService) GossipHandler() {
	for {
		msg := <-service.Messages
		log.Infof("Receive Gossip: %v\n", string(msg.data))
		var neg = pb.Negotiate{}
		err := proto.Unmarshal(msg.data, &neg)
		if err != nil {
			log.Errorf("Unmarshall gossip error: %v", err)
			continue
		}
		clientInfo, ok := service.network.proxyService.ClientMap[neg.Destination]
		if !ok {
			log.Info("Gossip not for my client")
			return
		}
		go service.handleGossipMsg(clientInfo, &neg)
	}
}

func (service *PubsubService) handleGossipMsg(clientInfo clientInfo, neg *pb.Negotiate) {
	if _, ok := service.network.SessionMapper.SessionmapID[neg.SessionId]; ok {
		log.Warnf("session already exist %v", neg.SessionId)
	}

	joinNode, err := peer.Decode(neg.Join)
	var relayId core.PeerID

	if joinNode == service.network.host.ID() {
		log.Info("act as both joint and exit")
	} else {
		for _, id := range service.network.host.Peerstore().Peers() {
			//todo:scheme to select relay node
			if id != joinNode {
				relayId = id
				break
			}
		}
		//todo：process error (retry)
		//new session to relay
		err = service.network.NewSessionToPeer(relayId, neg.SessionId, ExitRole, RelayRole)
		if err != nil {
			log.Errorf("New session to relay err %v", err)
			return
		}

		err = service.network.ExpendSession(relayId, joinNode, neg.SessionId)
		if err != nil {
			log.Errorf("Expend session err %v", err)
			return
		}
	}
	//new session to receiver
	err = service.network.NewSessionToPeer(clientInfo.peerID, neg.SessionId, ExitRole, AnswerRole)
	if err != nil {
		log.Errorf("New session to destination err:%v", err)
		return
	}
}
