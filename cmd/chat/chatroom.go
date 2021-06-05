package chat

import (
	"context"
	"whitenoise/common/log"
	"whitenoise/protocol/relay"
	"whitenoise/sdk"
	"whitenoise/secure"
)

type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *relay.RelayMsg
	wnsdk    *sdk.WhiteNoiseClient
	ctx      context.Context
	chatUI   *ChatUI
	roomName string
	self     string
	PeerList []string
	nick     string
}

// ChatMessage gets converted to/from JSON and sent in the body of pubsub messages.
type ChatMessageRoom struct {
	Message    string
	SenderID   string
	SenderNick string
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinChatRoom(ctx context.Context, nick string, selfID string, roomName string, wnsdk *sdk.WhiteNoiseClient) (*ChatRoom, error) {
	cr := &ChatRoom{
		ctx:      ctx,
		self:     selfID,
		nick:     nick,
		roomName: roomName,
		Messages: make(chan *relay.RelayMsg),
		wnsdk:    wnsdk,
	}
	if roomName == "" {
		err := wnsdk.EventBus().SubscribeOnce(sdk.GetCircuitTopic, func(sessionID string) {
			cr.roomName = sessionID
		})
		if err != nil {
			return nil, err
		}
	}

	// start reading messages from the subscription in a loop
	return cr, nil
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *ChatRoom) readLoop() {
	for {
		conn, ok := cr.chatUI.service.GetCircuit(cr.roomName)
		if !ok {
			continue
		}
		msg, err := secure.ReadPayload(conn)
		log.Debug("get msg:", msg)
		if err != nil {
			log.Error("readloop", err)
			continue
		}
		cr.Messages <- (*relay.RelayMsg)(&msg)
	}
}
