package relay

import (
	"bytes"
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/magiconair/properties/assert"
	"testing"
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/secure"
)

func TestCircuitConn(t *testing.T) {
	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)
	circuit := CircuitConn{
		buffer: SafeBuffer{
			b:           new(bytes.Buffer),
			eventBus:    EventBus.New(),
			readTimeout: common.UnreadableTimeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
	msg := []byte("hello whitenoise")
	circuit.InboundMsg(secure.EncodePayload(msg))
	recMsg, err := secure.ReadPayload(&circuit)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, msg, recMsg)
}
