package relay

import (
	"bytes"
	"context"
	"github.com/asaskevich/EventBus"
	"io"
	"net"
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

//todo:remove the lock, use buffer pool
type SafeBuffer struct {
	b           *bytes.Buffer
	mut         sync.RWMutex
	eventBus    EventBus.Bus
	readTimeout time.Duration
}

//Now we use most simple polling way to read synchronously.
//todo: change into a more efficient way.
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

func (c *CircuitConn) LocalAddr() net.Addr {
	return nil
}

func (c *CircuitConn) RemoteAddr() net.Addr {
	return nil
}

func (c *CircuitConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *CircuitConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *CircuitConn) SetWriteDeadline(t time.Time) error {
	return nil
}
