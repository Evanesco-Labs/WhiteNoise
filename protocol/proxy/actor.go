package proxy

import (
	"github.com/AsynkronIT/protoactor-go/actor"
)

type ReqGetClient struct {
	Destination string
}

type ResGetClient struct {
	Info ClientInfo
	Ok   bool
}

func (manager *ProxyManager) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ReqGetClient:
		info, ok := manager.GetClient(msg.Destination)
		ctx.Respond(ResGetClient{
			Info: info,
			Ok:   ok,
		})
	default:
		//log.Debugf("Proxy actor cannot handle request %v", msg)
	}
}
