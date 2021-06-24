package relay

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/common/config"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
	"github.com/Evanesco-Labs/WhiteNoise/network/session"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/ack"
	"github.com/Evanesco-Labs/WhiteNoise/secure"
)

const RelayProtocol string = "/relay"

func (manager *RelayMsgManager) RelayStreamHandler(stream network.Stream) {
	log.Debug("Got a new stream: ", stream.ID())
	str := session.NewStream(stream, manager.context)
	manager.AddStream(str)
	go manager.RelayInboundHandler(str)
}

func (manager *RelayMsgManager) RelayInboundHandler(s session.Stream) {
	for {
		select {
		case <-s.Ctx.Done():
			log.Debugf("stream %v context done", s.StreamId)
			return
		default:
			break
		}
		//todo:release msgBytes (cautious rw overlap)
		msgBytes, err := s.RW.ReadMsg()
		if err != nil {
			//todo:clean closed stream
			log.Infof("stream %v read err %v", s.StreamId, err)
			s.Close()
			return
		}

		var relay = pb.Relay{}
		err = proto.Unmarshal(msgBytes, &relay)
		if err != nil {
			log.Error("unmarshal err", err)
			continue
		}

		switch relay.Type {
		case pb.Relaytype_SetSessionId:
			err = manager.handleSetSession(&relay, s)
			if err != nil {
				log.Warn("Handle set session command err ", err)
			}
			continue

		case pb.Relaytype_Data:
			err = manager.handleRelayMsg(&relay, s, msgBytes)
			if err != nil {
				log.Error("Handle relay message err ", err)
			}
			continue

		case pb.Relaytype_Probe:
			err = manager.handleRelayProbe(&relay, s, msgBytes)
			if err != nil {
				log.Error("Handle relay probe err", err)
			}
			continue

		case pb.Relaytype_Disconnect:
			err = manager.handleDisconnect(&relay, s, msgBytes)
			if err != nil {
				log.Error("Handle disconnect err", err)
			}
		case pb.Relaytype_Ack:
		case pb.Relaytype_Wake:
			log.Debug("Stream awake")
		case pb.Relaytype_Success:
			log.Debug("Receive circuit success signal")
			go func() {
				err := manager.handleCircuitSuccess(&relay, s, msgBytes)
				if err != nil {
					log.Error("Handle circuit success signal err", err)
				}
			}()
			continue
		default:
			log.Warn("Got relay error type ")
		}
	}
}

func (manager *RelayMsgManager) handleSetSession(relay *pb.Relay, s session.Stream) error {
	var ackMsg = pb.Ack{
		CommandId: relay.Id,
		Result:    false,
		Data:      []byte{},
	}

	var setSession pb.SetSessionIdMsg
	err := proto.Unmarshal(relay.Data, &setSession)
	if err != nil {
		ackMsg.Data = []byte("setSessionIdMsg unmarshall err " + err.Error())
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: s.RemotePeer})
		return errors.New("setSessionIdMsg unmarshall err " + err.Error())
	}

	//Client reject Sessions for Server Role
	if manager.role == config.ClientMode && common.SessionRole(setSession.Role) != common.AnswerRole {
		ackMsg.Data = []byte("reject")
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: s.RemotePeer})
		return errors.New("reject")
	}

	if manager.role == config.ClientMode && common.SessionRole(setSession.Role) == common.AnswerRole {
		manager.AddCircuitConnAnswer(setSession.SessionId)
	}

	//act as both entry and relay
	//1. tell join node stream closed 2. remove the stream to joint node 3. add stream to exit node 4. response success to clients 5. ignore session expend command
	sess, ok := manager.GetSession(setSession.SessionId)
	if ok && sess.Role == common.EntryRole && common.SessionRole(setSession.Role) == common.RelayRole {
		log.Warnf("act as both entry and relay for session %v", setSession.SessionId)
		if sess.IsReady() {
			//tell join node to close circuit
			streamJoin := sess.Pair[1]
			disData, _ := NewDisconnect(setSession.SessionId)
			streamJoin.RW.WriteMsg(disData)
			//remove the stream to joint node
			manager.DeleteStream(streamJoin.StreamId)
			sess.Pair[1] = s
			manager.AddSessionId(setSession.SessionId, sess)
			manager.AddStream(s)
			manager.AddStreamSessionID(s.StreamId, setSession.SessionId)
			log.Debugf("add sessionid %v to stream %v\n", setSession.SessionId, s.StreamId)
			log.Debugf("session: %v\n", sess)
			//respond ack to exit node
			ackMsg.Result = true
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: s.RemotePeer})
			return nil
		} else {
			manager.CloseCircuit(setSession.SessionId)
			return errors.New("create circuit as entry node failed")
		}
	}

	if !ok {
		sess = session.NewSession()
		sess.SetSessionID(setSession.SessionId)
		sess.Role = common.SessionRole(setSession.Role)
	}
	sess.AddStream(s)
	manager.AddSessionId(setSession.SessionId, sess)
	manager.AddStream(s)
	manager.AddStreamSessionID(s.StreamId, setSession.SessionId)

	log.Debugf("add sessionId %v to stream %v\n", setSession.SessionId, s.StreamId)
	log.Debugf("session: %v\n", sess)

	ackMsg.Result = true
	manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: s.RemotePeer})
	return nil
}

func (manager *RelayMsgManager) handleRelayMsg(relay *pb.Relay, s session.Stream, data []byte) error {
	var relayMsg pb.RelayMsg
	err := proto.Unmarshal(relay.Data, &relayMsg)
	if err != nil {
		return err
	}

	sess, ok := manager.GetSession(relayMsg.SessionId)
	if !ok {
		log.Warn("relay no such session")
		return nil
	}
	if sess.Role == common.CallerRole || sess.Role == common.AnswerRole {
		if c, ok := manager.GetCircuit(relayMsg.SessionId); ok {
			c.InboundMsg(relayMsg.Data)
		} else {
			log.Warnf("Got relay msg, but session %v have not init CircuitConn in MsgManager", relayMsg.SessionId)
		}
		log.Debugf("Got msg from circuit %v: %v\n", relayMsg.SessionId, string(relayMsg.Data))
		return nil
	}
	if sess.IsReady() {
		part, err := sess.GetPattern(s.StreamId)
		if err != nil {
			manager.CloseCircuit(relayMsg.SessionId)
			log.Error("get part err", err)
			return err
		}
		err = part.RW.WriteMsg(data)
		if err != nil {
			manager.CloseCircuit(relayMsg.SessionId)
			log.Error("write err", err)
			return err
		}
	} else {
		log.Warnf("Session not ready yet %v", relayMsg.SessionId)
	}
	return nil
}

func (manager *RelayMsgManager) handleRelayProbe(relay *pb.Relay, s session.Stream, data []byte) error {
	var probe pb.ProbeSignal
	err := proto.Unmarshal(relay.Data, &probe)
	if err != nil {
		return err
	}
	sess, ok := manager.GetSession(probe.SessionId)
	if ok && sess.Role == common.JointRole {
		sessionProbe := session.Probe{
			SessionId: probe.SessionId,
			Rand:      probe.Data,
		}
		v, ok := manager.probeMap.Load(sessionProbe.SessionId)
		if !ok {
			manager.probeMap.Store(sessionProbe.SessionId, sessionProbe)
			return nil
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
		return nil
	}
	if sess.IsReady() {
		part, err := sess.GetPattern(s.StreamId)
		if err != nil {
			log.Error("get part err", err)
			return err
		}
		err = part.RW.WriteMsg(data)
		if err != nil {
			log.Error("write err", err)
			return err
		}
	}
	return nil
}

func (manager *RelayMsgManager) handleDisconnect(relay *pb.Relay, s session.Stream, data []byte) error {
	log.Debug("Handle Disconnect signal")
	var dis pb.Disconnect
	err := proto.Unmarshal(relay.Data, &dis)
	if err != nil {
		return err
	}

	log.Infof("Close circuit %v", dis.SessionId)
	defer func() { manager.RemoveSession(dis.SessionId) }()
	_, ok := manager.sessionMap.Load(dis.SessionId)
	if !ok {
		return errors.New("no such session")
	}

	err = manager.ForwardRelay(dis.SessionId, data, s.RemotePeer)
	if err != nil {
		return err
	}
	return nil
}

func (manager *RelayMsgManager) handleCircuitSuccess(relay *pb.Relay, s session.Stream, data []byte) error {
	var succ pb.CircuitSuccess
	if relay.Data == nil {
		log.Error("nil data")
	}
	err := proto.Unmarshal(relay.Data, &succ)
	if err != nil {
		return err
	}
	sess, ok := manager.GetSession(succ.SessionId)
	if !ok {
		return errors.New("no such session " + succ.SessionId)
	}
	if sess.Role == common.CallerRole {
		log.Debug("caller handle circuit success")
		v, ok := manager.circuitConnMap.Load(succ.SessionId)
		if !ok {
			return errors.New("circuitConn not exist")
		}
		conn := v.(*CircuitConn)
		conn.state = CircuitConnReady
		manager.circuitConnMap.Store(succ.SessionId, conn)
		log.Debug("circuitConn ready ", succ.SessionId)
		err := manager.NewSecureConnCaller(conn)
		if err != nil {
			log.Error(err)
		}
		return nil
	}

	if sess.Role == common.AnswerRole {
		v, ok := manager.circuitConnMap.Load(succ.SessionId)
		if !ok {
			return errors.New("circuitConn not exist")
		}
		conn := v.(*CircuitConn)
		conn.state = CircuitConnReady
		manager.circuitConnMap.Store(succ.SessionId, conn)
		log.Debug("circuitConn ready ", succ.SessionId)
		err := manager.NewSecureConnAnswer(conn)
		if err != nil {
			log.Error(err)
		}
		return nil
	}

	if !sess.IsReady() {
		return errors.New("Received success signal,but session not ready: " + succ.SessionId)
	}

	part, err := sess.GetPattern(s.StreamId)
	if err != nil {
		log.Error("get part err", err)
		return err
	}
	err = part.RW.WriteMsg(data)
	if err != nil {
		log.Error("write err", err)
		return err
	}
	return nil
}

func NewSetSessionIDCommand(sessionID string, otherRole common.SessionRole) ([]byte, string) {
	cmd := pb.SetSessionIdMsg{
		SessionId: sessionID,
		Role:      int32(otherRole),
	}
	data, _ := proto.Marshal(&cmd)
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_SetSessionId,
		Data: data,
	}
	noId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(noId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	comd, _ := proto.Marshal(&relay)
	return comd, relay.Id
}

func NewRelayWake() []byte {
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Wake,
		Data: []byte{},
	}
	relayNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(relayNoId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	relayBytes, _ := proto.Marshal(&relay)
	return relayBytes
}

func NewRelayMsg(msg []byte, sessionID string) []byte {
	relayMsg := pb.RelayMsg{
		SessionId: sessionID,
		Data:      msg,
	}
	relayMsgData, _ := proto.Marshal(&relayMsg)
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Data,
		Data: relayMsgData,
	}
	relayNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(relayNoId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	relayBytes, _ := proto.Marshal(&relay)
	return relayBytes
}

func NewProbeSignal(sessionId string) ([]byte, error) {
	signal := sha256.Sum256([]byte(sessionId))
	probe := pb.ProbeSignal{
		SessionId: sessionId,
		Data:      signal[:],
	}
	data, err := proto.Marshal(&probe)
	if err != nil {
		return nil, err
	}
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Probe,
		Data: data,
	}
	dataNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(dataNoId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	relayData, err := proto.Marshal(&relay)
	if err != nil {
		return nil, err
	}
	return relayData, nil
}

func NewDisconnect(sessionId string) ([]byte, error) {
	dis := pb.Disconnect{
		SessionId: sessionId,
		ErrCode:   0,
	}
	data, err := proto.Marshal(&dis)
	if err != nil {
		return nil, err
	}
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Disconnect,
		Data: data,
	}
	dataNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(dataNoId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	relayData, err := proto.Marshal(&relay)
	if err != nil {
		return nil, err
	}
	return relayData, nil
}

func NewCircuitSuccess(sessionId string) []byte {
	succ := pb.CircuitSuccess{SessionId: sessionId}
	data, _ := proto.Marshal(&succ)
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Success,
		Data: data,
	}
	dataNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(dataNoId)
	relay.Id = secure.EncodeMSGIDHash(hash[:])
	relayData, _ := proto.Marshal(&relay)
	return relayData
}
