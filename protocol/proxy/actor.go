package proxy

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ReqGetClient struct {
	Destination string
}

type ResGetClient struct {
	Info ClientInfo
	Ok   bool
}

type ReqDecrypt struct {
	CipherText []byte
	Des        peer.ID
}

type ResDecrypt struct {
	Join      string
	SessionId string
	Err       error
}

type ReqUnregister struct {
	PeerId peer.ID
}

func (manager *ProxyManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ReqGetClient:
		info, ok := manager.GetClient(msg.Destination)
		ctx.Respond(ResGetClient{
			Info: info,
			Ok:   ok,
		})
	case ReqDecrypt:
		sessionId, join, err := manager.DecryptGossip(msg.Des, msg.CipherText)
		ctx.Respond(ResDecrypt{
			Join:      join,
			SessionId: sessionId,
			Err:       err,
		})
	case ReqUnregister:
		manager.UnRegisterClient(msg.PeerId)
	default:
		//log.Debugf("Proxy actor cannot handle request %v", msg)
	}
}
