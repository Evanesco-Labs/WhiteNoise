package actorMsg

import core "github.com/libp2p/go-libp2p-core"

type ReqGossipJoint struct {
	DesHash   string
	Joint     core.PeerID
	SessionId string
}

type ResError struct {
	Err error
}
