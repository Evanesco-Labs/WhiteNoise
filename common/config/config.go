package config

type ServiceMode int

const ServerMode ServiceMode = 0
const ClientMode ServiceMode = 1
const BootMode ServiceMode = 2

type NetworkConfig struct {
	RendezvousString string
	ListenHost       string
	ListenPort       int
	BootStrapPeers   string
	Mode             ServiceMode
}
