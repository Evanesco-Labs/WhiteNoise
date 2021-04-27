package actorMsg

import "github.com/libp2p/go-libp2p-core/peer"

type ReqGossipJoint struct {
	DesHash   string
	NegCypher []byte
}

type ReqDHTPeers struct {
	Max int
}

type ResDHTPeers struct {
	PeerInfos []peer.AddrInfo
}

type ResError struct {
	Err error
}
