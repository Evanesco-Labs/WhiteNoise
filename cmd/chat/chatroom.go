package chat

import (
	"context"
	"github.com/libp2p/go-libp2p-core/peer"
	"whitenoise/common/log"
	"whitenoise/protocol/relay"
	"whitenoise/secure"
)

type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *relay.RelayMsg

	ctx      context.Context
	chatUI   *ChatUI
	roomName string
	self     peer.ID
	PeerList []peer.ID
	nick     string
}

// ChatMessage gets converted to/from JSON and sent in the body of pubsub messages.
type ChatMessage struct {
	Message    string
	SenderID   string
	SenderNick string
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinChatRoom(ctx context.Context, selfID peer.ID, nickname string, roomName string) (*ChatRoom, error) {
	cr := &ChatRoom{
		ctx:      ctx,
		self:     selfID,
		nick:     nickname,
		roomName: roomName,
		Messages: make(chan *relay.RelayMsg),
	}

	// start reading messages from the subscription in a loop
	return cr, nil
}

func (cr *ChatRoom) ListPeers() []peer.ID {
	var peers []peer.ID
	for _, session := range cr.chatUI.service.NoiseService.Relay().SessionMap() {
		for _, pair := range session.GetPair() {
			peers = append(peers, pair.RemotePeer)
		}
	}
	return peers
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *ChatRoom) readLoop() {
	//if _, ok := cr.chatUI.service.MsgManager.Circuits[cr.roomName]; !ok {
	//	cr.chatUI.service.MsgManager.Circuits[cr.roomName] = make(chan whitenoise.RelayMsg)
	//}
	//cr.chatUI.service.NoiseService.Relay().AddMsgChan(cr.roomName)
	for {
		//msgChan, ok := cr.chatUI.service.NoiseService.Relay().GetMsgChan(cr.roomName)
		//if !ok {
		//	continue
		//}
		//select {
		//case m := <-msgChan:
		//	cr.Messages <- &m
		//}
		conn, ok := cr.chatUI.service.NoiseService.Relay().GetSecureConn(cr.roomName)
		if !ok {
			continue
		}
		msg, err := secure.ReadPayload(conn)
		if err != nil {
			log.Error("readloop", err)
			continue
		}
		cr.Messages <- (*relay.RelayMsg)(&msg)
	}
}

func topicName(roomName string) string {
	return "chat-room:" + roomName
}
