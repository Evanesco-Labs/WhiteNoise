package relay

import (
	"bytes"
	"context"
	"github.com/asaskevich/EventBus"
	"io"
	"sync"
	"time"
	"whitenoise/common"
	"whitenoise/common/account"
	"whitenoise/common/log"
)

type CircuitConnState int

const (
	CircuitConnBuilding CircuitConnState = iota
	CircuitConnReady
)

const (
	NonEmptyTopic = "NonEmptyTopic"
	EmptyTopic    = "EmptyTopic"
	FullTopic     = "FullTopic"
	NonFullTopic  = "NonFullTopic"
)

type SafeBuffer struct {
	b           *bytes.Buffer
	mut         sync.RWMutex
	eventBus    EventBus.Bus
	readTimeout time.Duration
}

//Using event bus to read synchronously.
func (b *SafeBuffer) Read(p []byte) (n int, err error) {
	n, err = b.b.Read(p)
	if err != nil {
		if err == io.EOF {
			wait := make(chan struct{})
			timeout := time.After(b.readTimeout)
			b.eventBus.SubscribeOnce(NonEmptyTopic, func() {
				n, err = b.b.Read(p)
				wait <- struct{}{}
			})
			select {
			case <-wait:
			case <-timeout:
				log.Error("connection unreadable timeout")
			}
		}
		return n, err
	}
	return n, nil
}

func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	return b.b.Write(p)
}

func (b *SafeBuffer) SetReadTimeout(duration time.Duration) {
	b.readTimeout = duration
}

type CircuitConn struct {
	localWhiteNoiseID  account.WhiteNoiseID
	remoteWhiteNoiseId account.WhiteNoiseID
	buffer             SafeBuffer
	sessionId          string
	relayMananger      *RelayMsgManager
	ctx                context.Context
	cancel             context.CancelFunc
	state              CircuitConnState
}

func (manager *RelayMsgManager) NewCircuitConn(parentCtx context.Context, sessionID string, remote account.WhiteNoiseID) *CircuitConn {
	ctx, cancel := context.WithCancel(parentCtx)
	circuit := CircuitConn{
		localWhiteNoiseID:  manager.Account.GetWhiteNoiseID(),
		remoteWhiteNoiseId: remote,
		buffer: SafeBuffer{
			b:           new(bytes.Buffer),
			eventBus:    EventBus.New(),
			readTimeout: common.UnreadableTimeout,
		},
		relayMananger: manager,
		ctx:           ctx,
		cancel:        cancel,
		sessionId:     sessionID,
		state:         CircuitConnBuilding,
	}
	return &circuit
}

func (c *CircuitConn) State() CircuitConnState {
	return c.state
}

func (c *CircuitConn) Read(b []byte) (n int, err error) {
	return c.buffer.Read(b)
}

func (c *CircuitConn) Write(b []byte) (int, error) {
	msg := NewRelayMsg(b, c.sessionId)
	err := c.relayMananger.SendRelay(c.sessionId, msg)
	return len(b), err
}

func (c *CircuitConn) InboundMsg(b []byte) {
	_, err := c.buffer.Write(b)
	c.buffer.eventBus.Publish(NonEmptyTopic)
	if err != nil {
		log.Error("inbound msg buffer write", err)
	}
}

func (c *CircuitConn) Close() error {
	c.cancel()
	return nil
}

func (c *CircuitConn) LocalID() account.WhiteNoiseID {
	return c.localWhiteNoiseID
}

func (c *CircuitConn) RemoteID() account.WhiteNoiseID {
	return c.remoteWhiteNoiseId
}

func (c *CircuitConn) GetSessionID() string {
	return c.sessionId
}
