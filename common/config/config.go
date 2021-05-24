package config

import "github.com/libp2p/go-libp2p-core/peer"

var WhiteListPeers map[peer.ID]bool

type ServiceMode int

const ServerMode ServiceMode = 0
const ClientMode ServiceMode = 1
const BootMode ServiceMode = 2

type YmlConfig struct {
	Whitelist []string
}

type NetworkConfig struct {
	RendezvousString string
	ListenHost       string
	ListenPort       int
	BootStrapPeers   string
	Mode             ServiceMode
	WhiteList        bool
}
