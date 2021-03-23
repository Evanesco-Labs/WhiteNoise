package relay

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"whitenoise/secure"
)

func TestCircuitConn(t *testing.T) {
	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)
	circuit := CircuitConn{
		localPeerId:  "",
		remotePeerId: "",
		buffer: SafeBuffer{
			b: new(bytes.Buffer),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	circuit.InboundMsg(secure.EncodePayload([]byte("hello")))
	msg, err := secure.ReadPayload(&circuit)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(msg))
}
