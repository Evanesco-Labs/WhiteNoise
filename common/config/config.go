package config

type ServiceMode int

const ServerMode ServiceMode = 0
const ClientMode ServiceMode = 1

type NetworkConfig struct {
	RendezvousString string
	ProtocolID       string
	ListenHost       string
	ListenPort  int
	BootStrapPeers   string
	Mode             ServiceMode
}

//
//func NewConfig() *NetworkConfig {
//	return parseFlags()
//}
//
//func parseFlags() *NetworkConfig {
//	c := &NetworkConfig{}
//	flag.StringVar(&c.RendezvousString, "rendezvous", "meetme", "Unique string to identify group of nodes. Share this with your friends to let them connect with you")
//	flag.StringVar(&c.ListenHost, "host", "127.0.0.1", "The bootstrap node host listen address\n")
//	flag.StringVar(&c.ProtocolID, "pid", "whitenoise", "Sets a protocol id for stream headers")
//	flag.IntVar(&c.ListenPort, "port", 4001, "node listen port")
//	flag.Parse()
//	return c
//}
