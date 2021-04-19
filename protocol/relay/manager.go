package relay

import (
	"bytes"
	"context"
	"errors"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/asaskevich/EventBus"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
	"sync"
	"time"
	"whitenoise/common"
	"whitenoise/common/account"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/network/session"
	"whitenoise/protocol/ack"
	"whitenoise/secure"
)

type RelayMsg []byte

type StreamInfo struct {
	stream    session.Stream
	sessionID string
}

type RelayMsgManager struct {
	mut               sync.Mutex
	circuitConnMap    map[string]*CircuitConn
	secureConnMap     map[string]*secure.SecureSession
	streamMap         map[string]StreamInfo
	sessionMap        map[string]session.Session
	probeMap          map[string]session.Probe
	ProbeChan         chan session.Probe
	ackPid            *actor.PID
	relayPid          *actor.PID
	host              core.Host
	actorCtx          *actor.RootContext
	context           context.Context
	role              config.ServiceMode
	privateKey        crypto.PrivKey
	SetSessionTimeout time.Duration
	Account           account.Account
	eb                EventBus.Bus
}

func NewRelayMsgManager(host core.Host, ctx context.Context, actCtx *actor.RootContext, role config.ServiceMode,
	privateKey crypto.PrivKey, acc account.Account, eb EventBus.Bus) *RelayMsgManager {
	return &RelayMsgManager{
		circuitConnMap:    make(map[string]*CircuitConn),
		secureConnMap:     make(map[string]*secure.SecureSession),
		streamMap:         make(map[string]StreamInfo),
		sessionMap:        make(map[string]session.Session),
		probeMap:          make(map[string]session.Probe),
		ProbeChan:         make(chan session.Probe),
		host:              host,
		actorCtx:          actCtx,
		context:           ctx,
		role:              role,
		SetSessionTimeout: common.SetSessionTimeout,
		privateKey:        privateKey,
		Account:           acc,
		eb:                eb,
	}
}

func (manager *RelayMsgManager) Start() {
	props := actor.PropsFromProducer(func() actor.Actor {
		return manager
	})
	go manager.RunProbeHandler(manager.context)
	manager.relayPid = manager.actorCtx.Spawn(props)
}

func (manager *RelayMsgManager) Pid() *actor.PID {
	return manager.relayPid
}

func (manager *RelayMsgManager) SetPid(ackPid *actor.PID) {
	manager.ackPid = ackPid
}

func (manager *RelayMsgManager) RemoveSession(sessionId string) {
	manager.mut.Lock()
	defer manager.mut.Unlock()

	if secureConn, ok := manager.secureConnMap[sessionId]; ok {
		secureConn.Close()
		delete(manager.secureConnMap, sessionId)
	}

	if conn, ok := manager.circuitConnMap[sessionId]; ok {
		conn.Close()
		delete(manager.circuitConnMap, sessionId)
	}

	if sess, ok := manager.sessionMap[sessionId]; ok {
		for _, stream := range sess.Pair {
			stream.Close()
		}
		delete(manager.sessionMap, sessionId)
	}
	delete(manager.probeMap, sessionId)

}

func (manager *RelayMsgManager) GetCircuit(sessionId string) (*CircuitConn, bool) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	c, ok := manager.circuitConnMap[sessionId]
	return c, ok
}

func (manager *RelayMsgManager) GetSecureConn(sessionId string) (*secure.SecureSession, bool) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	c, ok := manager.secureConnMap[sessionId]
	return c, ok
}

func (manager *RelayMsgManager) AddCircuitConnCaller(sessionId string, remote account.WhiteNoiseID) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	_, ok := manager.circuitConnMap[sessionId]
	if !ok {
		conn := manager.NewCircuitConn(manager.context, sessionId, remote)
		manager.circuitConnMap[sessionId] = conn
	}
}

func (manager *RelayMsgManager) AddCircuitConnAnswer(sessionId string) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	_, ok := manager.circuitConnMap[sessionId]
	if !ok {
		conn := manager.NewCircuitConn(manager.context, sessionId, []byte{})
		manager.circuitConnMap[sessionId] = conn
	}
}

func (manager *RelayMsgManager) RunProbeHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}
		probe := <-manager.ProbeChan
		p, ok := manager.probeMap[probe.SessionId]
		if !ok {
			manager.probeMap[probe.SessionId] = probe
			continue
		}
		if bytes.Equal(p.Rand, probe.Rand) {
			//circuit success
			log.Debug("send circuit success signal")
			data := NewCircuitSuccess(probe.SessionId)
			err := manager.SendRelay(probe.SessionId, data)
			if err != nil {
				log.Error(err)
			}
		} else {
			manager.CloseCircuit(probe.SessionId)
		}
	}
}

// WaitForProbe todo: add join node time out when building a circuit
func (manager *RelayMsgManager) WaitForProbe() {}

func (manager *RelayMsgManager) AddStream(s session.Stream) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	manager.streamMap[s.StreamId] = StreamInfo{
		stream:    s,
		sessionID: "",
	}
}

func (manager *RelayMsgManager) AddStreamSessionID(streamID string, sessionID string) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	if streamInfo, ok := manager.streamMap[streamID]; ok {
		streamInfo.sessionID = sessionID
		manager.streamMap[streamID] = streamInfo
	}
}

func (manager *RelayMsgManager) GetSession(id string) (session.Session, bool) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	s, ok := manager.sessionMap[id]
	return s, ok
}

func (manager *RelayMsgManager) AddSessionId(id string, s session.Session) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	manager.sessionMap[id] = s
}

func (manager *RelayMsgManager) SetRole(sessionId string, role common.SessionRole) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	s, ok := manager.sessionMap[sessionId]
	if !ok {
		return
	}
	s.Role = role
	manager.sessionMap[sessionId] = s
}

func (manager *RelayMsgManager) GetStream(id string) (StreamInfo, bool) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	s, ok := manager.streamMap[id]
	return s, ok
}

func (manager *RelayMsgManager) DeleteStream(streamID string) {
	delete(manager.streamMap, streamID)
}

func (manager *RelayMsgManager) SessionMap() map[string]session.Session {
	return manager.sessionMap
}

func (manager *RelayMsgManager) SendRelay(sessionid string, data []byte) (err error) {
	defer func() {
		if err != nil {
			log.Errorf("SendRelay err: %v", err)
			manager.CloseCircuit(sessionid)
		}
	}()
	sess, ok := manager.GetSession(sessionid)
	if !ok {
		return errors.New("SendRelay no such session")
	}

	if len(sess.GetPair()) == 0 {
		return errors.New("stream pair in this session is empty")
	}
	for _, stream := range sess.GetPair() {
		err := stream.RW.WriteMsg(data)
		if err != nil {
			log.Error("write err", err)
			return err
		}
	}
	return nil
}

func (manager *RelayMsgManager) ForwardRelay(sessionId string, data []byte, from core.PeerID) (err error) {
	defer func() {
		if err != nil {
			log.Errorf("SendRelay err: %v", err)
			manager.CloseCircuit(sessionId)
		}
	}()
	sess, ok := manager.GetSession(sessionId)
	if !ok {
		return errors.New("SendRelay no such session")
	}
	if len(sess.GetPair()) == 0 {
		return errors.New("stream pair in this session is empty")
	}
	for _, stream := range sess.GetPair() {
		if stream.RemotePeer == from {
			continue
		}
		err := stream.RW.WriteMsg(data)
		if err != nil {
			log.Error("write err", err)
			return err
		}
	}
	return nil
}

func (manager *RelayMsgManager) SendDisconnectRelay(sessionId string) (err error) {
	disData, err := NewDisconnect(sessionId)
	if err != nil {
		return err
	}
	sess, ok := manager.GetSession(sessionId)
	if !ok {
		return errors.New("SendRelay no such session")
	}

	if len(sess.GetPair()) == 0 {
		return errors.New("stream pair in this session is empty")
	}
	for _, stream := range sess.GetPair() {
		err := stream.RW.WriteMsg(disData)
		if err != nil {
			log.Debug("write err", err)
			continue
		}
	}
	return nil
}

func (manager *RelayMsgManager) CloseCircuit(sessionId string) error {
	log.Infof("Close circuit %v", sessionId)
	defer func() { manager.RemoveSession(sessionId) }()
	_, ok := manager.sessionMap[sessionId]
	if !ok {
		return errors.New("no such session")
	}
	err := manager.SendDisconnectRelay(sessionId)
	if err != nil {
		return err
	}
	return nil
}

func (manager *RelayMsgManager) NewRelayStream(peerID core.PeerID) (string, error) {
	stream, err := manager.host.NewStream(manager.context, peerID, protocol.ID(RelayProtocol))
	if err != nil {
		log.Infof("generate new stream to %v error: %v\n", peerID, err)
		return "", err
	}
	log.Info("gen new stream: ", stream.ID())
	s := session.NewStream(stream, manager.context)
	manager.AddStream(s)
	go manager.RelayInboundHandler(s)
	log.Infof("Connected to:%v \n", peerID)
	err = s.RW.WriteMsg(NewRelayWake())
	if err != nil {
		log.Error("write err", err)
		return "", err
	}
	return s.StreamId, nil
}

func (manager *RelayMsgManager) NewSessionToPeer(peerID core.PeerID, sessionID string, myRole common.SessionRole, otherRole common.SessionRole) error {
	streamId, err := manager.NewRelayStream(peerID)
	if err != nil {
		return err
	}
	err = manager.SetSessionId(sessionID, streamId, myRole, otherRole)
	if err != nil {
		return err
	}
	return nil
}

func (manager *RelayMsgManager) SetSessionId(sessionID string, streamID string, myRole common.SessionRole, otherRole common.SessionRole) error {
	streamInfo, ok := manager.GetStream(streamID)
	if !ok {
		return errors.New("no such stream:" + streamID)
	}
	stream := streamInfo.stream
	data, id := NewSetSessionIDCommand(sessionID, otherRole)
	err := stream.RW.WriteMsg(data)
	if err != nil {
		log.Error("write err", err)
		return err
	}

	res := ack.Task{
		Id:      id,
		Channel: make(chan ack.Result),
	}
	manager.actorCtx.Request(manager.ackPid, ack.ReqAddTask{T: res})
	defer manager.actorCtx.Request(manager.ackPid, ack.ReqDeleteTask{Id: res.Id})
	select {
	case <-time.After(manager.SetSessionTimeout):
		err := errors.New("timeout")
		return err
	case result := <-res.Channel:
		if result.Ok {
			s, ok := manager.GetSession(sessionID)
			if !ok {
				s = session.NewSession()
				s.SetSessionID(sessionID)
				s.Role = myRole
			}
			s.AddStream(stream)
			manager.AddSessionId(sessionID, s)
			manager.AddStreamSessionID(streamID, sessionID)
			log.Infof("session: %v\n", s)
			return nil
		} else {
			return errors.New("cmd rejected: " + string(result.Data))
		}
	}
}

func (manager *RelayMsgManager) GetSessionIDList() []string {
	sessionList := make([]string, 0)
	for id, _ := range manager.sessionMap {
		sessionList = append(sessionList, id)
	}
	return sessionList
}
