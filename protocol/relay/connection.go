package relay

import (
	"bytes"
	"context"
	core "github.com/libp2p/go-libp2p-core"
	"io"
	"net"
	"sync"
	"time"
	"whitenoise/common"
	"whitenoise/common/log"
)

type CircuitConnState int

const (
	CircuitConnBuilding CircuitConnState = iota
	CircuitConnReady
)

//todo:remove the lock, use buffer pool
type SafeBuffer struct {
	b   *bytes.Buffer
	mut sync.RWMutex
}

//Now we use most simple polling way to read synchronously.
//todo: change into a more efficient way.
func (b *SafeBuffer) Read(p []byte) (n int, err error) {
	//b.mut.RLock()
	//defer b.mut.RUnlock()
	for {
		n, err := b.b.Read(p)
		if err != nil {
			if err == io.EOF {
				time.Sleep(common.CircuitConnReadPollCycle)
				continue
			}
			return n, err
		}
		return n, nil
	}
}
func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	//b.mut.Lock()
	//defer b.mut.Unlock()
	return b.b.Write(p)
}

type CircuitConn struct {
	localPeerId   core.PeerID
	remotePeerId  core.PeerID
	buffer        SafeBuffer
	sessionId     string
	relayMananger *RelayMsgManager
	ctx           context.Context
	cancel        context.CancelFunc
	state         CircuitConnState
}

func (manager *RelayMsgManager) NewCircuitConn(parentCtx context.Context, sessionID string, remote core.PeerID) *CircuitConn {
	ctx, cancel := context.WithCancel(parentCtx)
	circuit := CircuitConn{
		localPeerId:  manager.host.ID(),
		remotePeerId: remote,
		buffer: SafeBuffer{
			b: new(bytes.Buffer),
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
	//n, err = c.buffer.Write(b)
	//if err != nil {
	//	return n, err
	//}
	msg := NewRelayMsg(b, c.sessionId)
	err := c.relayMananger.SendRelay(c.sessionId, msg)
	return len(b), err
}

func (c *CircuitConn) InboundMsg(b []byte) {
	//err is always nil
	n, err := c.buffer.Write(b)
	log.Debugf("inbound msg buffer write len %v\n", n)
	if err != nil {
		log.Error("inbound msg buffer write", err)
	}
}

func (c *CircuitConn) Close() error {
	//err := c.relayMananger.CloseCircuit(c.sessionId)
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
