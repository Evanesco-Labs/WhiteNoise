package host

import (
	"context"
	"github.com/Evanesco-Labs/WhiteNoise/common/account"
	"github.com/Evanesco-Labs/WhiteNoise/common/config"
	"testing"
)

func TestNewHost(t *testing.T) {
	acc := account.GetAccount(1)
	priv := acc.GetP2PPrivKey()
	cfg := config.NetworkConfig{
		RendezvousString: "whitenoise",
		ListenHost:       "127.0.0.1",
		ListenPort:       3331,
		BootStrapPeers:   "",
		Mode:             config.BootMode,
	}
	_, _, err := NewHost(context.Background(), &cfg, priv)
	if err != nil {
		t.Fatal(err)
	}
	select {}
}
