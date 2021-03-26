package whitenoise

import (
	"flag"
)

type NetworkConfig struct {
	RendezvousString string
	ProtocolID       string
	ListenHost       string
	ListenPort       int
	Session          string
	Relay            bool
	Pub              string
	Expend           string
	Des              string
	Join             string
	Proxy            string
	Circuit          string
	Client           bool
}

func NewConfig() *NetworkConfig {
	return parseFlags()
}

func parseFlags() *NetworkConfig {
	c := &NetworkConfig{}
	flag.StringVar(&c.ListenHost, "host", "127.0.0.1", "The bootstrap node host listen address\n")
	flag.StringVar(&c.ProtocolID, "pid", "whitenoise", "Sets a protocol id for stream headers")
	flag.IntVar(&c.ListenPort, "port", 4001, "node listen port")
	flag.Parse()
	return c
}
