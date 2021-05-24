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
	circuitConnMap    sync.Map
	secureConnMap     sync.Map
	streamMap         sync.Map
	sessionMap        sync.Map
	probeMap          sync.Map
	ackPid            *actor.PID
	relayPid          *actor.PID
	host              core.Host
	actorCtx          *actor.RootContext
	context           context.Context
	role              config.ServiceMode
	privateKey        crypto.PrivKey
	SetSessionTimeout time.Duration
	Account           *account.Account
	eb                EventBus.Bus
}

func NewRelayMsgManager(host core.Host, ctx context.Context, actCtx *actor.RootContext, role config.ServiceMode, privateKey crypto.PrivKey, acc *account.Account, eb EventBus.Bus) *RelayMsgManager {
	return &RelayMsgManager{
		circuitConnMap:    sync.Map{},
		secureConnMap:     sync.Map{},
		streamMap:         sync.Map{},
		sessionMap:        sync.Map{},
		probeMap:          sync.Map{},
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
	manager.relayPid = manager.actorCtx.Spawn(props)
}

func (manager *RelayMsgManager) Pid() *actor.PID {
	return manager.relayPid
}

func (manager *RelayMsgManager) SetPid(ackPid *actor.PID) {
	manager.ackPid = ackPid
}

func (manager *RelayMsgManager) RemoveSession(sessionId string) {
	if v, ok := manager.secureConnMap.Load(sessionId); ok {
		v.(*secure.SecureSession).Close()
		manager.secureConnMap.Delete(sessionId)
	}

	if v, ok := manager.circuitConnMap.Load(sessionId); ok {
		v.(*CircuitConn).Close()
		manager.circuitConnMap.Delete(sessionId)
	}

	if sess, ok := manager.sessionMap.Load(sessionId); ok {
		sess := sess.(session.Session)
		for _, stream := range sess.Pair {
			stream.Close()
		}
		manager.sessionMap.Delete(sessionId)
	}
	manager.probeMap.Delete(sessionId)

}

func (manager *RelayMsgManager) GetCircuit(sessionId string) (*CircuitConn, bool) {
	v, ok := manager.circuitConnMap.Load(sessionId)
	if !ok {
		return nil, ok
	}
	return v.(*CircuitConn), ok
}

func (manager *RelayMsgManager) GetSecureConn(sessionId string) (*secure.SecureSession, bool) {
	v, ok := manager.secureConnMap.Load(sessionId)
	if !ok {
		return nil, ok
	}
	return v.(*secure.SecureSession), ok
}

func (manager *RelayMsgManager) AddCircuitConnCaller(sessionId string, remote account.WhiteNoiseID) {
	_, ok := manager.circuitConnMap.Load(sessionId)
	if !ok {
		conn := manager.NewCircuitConn(manager.context, sessionId, remote)
		manager.circuitConnMap.Store(sessionId, conn)
	}
}

func (manager *RelayMsgManager) AddCircuitConnAnswer(sessionId string) {
	_, ok := manager.circuitConnMap.Load(sessionId)
	if !ok {
		conn := manager.NewCircuitConn(manager.context, sessionId, []byte{})
		manager.circuitConnMap.Store(sessionId, conn)
	}
}

func (manager *RelayMsgManager) handleProbe(sessionProbe session.Probe) {
	v, ok := manager.probeMap.Load(sessionProbe.SessionId)
	if !ok {
		manager.probeMap.Store(sessionProbe.SessionId, sessionProbe)
		return
	}
	p := v.(session.Probe)
	if bytes.Equal(p.Rand, sessionProbe.Rand) {
		//circuit success
		log.Debug("send circuit success signal")
		data := NewCircuitSuccess(sessionProbe.SessionId)
		err := manager.SendRelay(sessionProbe.SessionId, data)
		if err != nil {
			log.Error(err)
		}
	} else {
		manager.CloseCircuit(sessionProbe.SessionId)
	}
}

func (manager *RelayMsgManager) AddStream(s session.Stream) {
	manager.streamMap.Store(s.StreamId, StreamInfo{
		stream:    s,
		sessionID: "",
	})
}

func (manager *RelayMsgManager) AddStreamSessionID(streamID string, sessionID string) {
	if v, ok := manager.streamMap.Load(streamID); ok {
		streamInfo := v.(StreamInfo)
		streamInfo.sessionID = sessionID
		manager.streamMap.Store(streamID, streamInfo)
	}
}

func (manager *RelayMsgManager) GetSession(id string) (session.Session, bool) {
	s, ok := manager.sessionMap.Load(id)
	if !ok {
		return session.Session{}, ok
	}
	return s.(session.Session), ok
}

func (manager *RelayMsgManager) AddSessionId(id string, s session.Session) {
	manager.sessionMap.Store(id, s)
}

func (manager *RelayMsgManager) SetRole(sessionId string, role common.SessionRole) {
	v, ok := manager.sessionMap.Load(sessionId)
	if !ok {
		return
	}
	sess := v.(session.Session)
	sess.Role = role
	manager.sessionMap.Store(sessionId, sess)
}

func (manager *RelayMsgManager) GetStream(id string) (StreamInfo, bool) {
	v, ok := manager.streamMap.Load(id)
	if !ok {
		return StreamInfo{}, ok
	}
	return v.(StreamInfo), ok
}

func (manager *RelayMsgManager) DeleteStream(streamID string) {
	manager.streamMap.Delete(streamID)
}

func (manager *RelayMsgManager) SessionMap() *sync.Map {
	return &manager.sessionMap
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
	_, ok := manager.sessionMap.Load(sessionId)
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
		log.Infof("newstream to %v error: %v\n", peerID, err)
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
	manager.sessionMap.Range(func(key, value interface{}) bool {
		sessionList = append(sessionList, key.(string))
		return true
	})
	return sessionList
}
