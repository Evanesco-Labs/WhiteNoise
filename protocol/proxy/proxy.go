package proxy

import (
	"context"
	cr "crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/asaskevich/EventBus"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"math/rand"
	"sync"
	"time"
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/common/account"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/crypto"
	"github.com/Evanesco-Labs/WhiteNoise/internal/actorMsg"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
	"github.com/Evanesco-Labs/WhiteNoise/network/session"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/ack"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/relay"
	"github.com/Evanesco-Labs/WhiteNoise/secure"
)

const PROXY_PROTOCOL string = "/proxy"
const ProxySerivceTime time.Duration = time.Hour

type ProxyManager struct {
	clientWNMap          sync.Map
	clientPeerMap        sync.Map
	actorCtx             *actor.RootContext
	ctx                  context.Context
	ackPid               *actor.PID
	host                 core.Host
	gossipPid            *actor.PID
	relayPid             *actor.PID
	proxyPid             *actor.PID
	RegisterProxyTimeout time.Duration
	NewCircuitTimeout    time.Duration
	DecryptReqTimeout    time.Duration
	RetryTimes           int
	//todo:clean tasks
	circuitTask sync.Map
	Account     *account.Account
	eb          EventBus.Bus
}

type ClientInfo struct {
	WhiteNoiseID crypto.WhiteNoiseID
	PeerID       core.PeerID
	state        int
	time         time.Duration
}

func NewProxyService(host core.Host, ctx context.Context, actCtx *actor.RootContext, acc *account.Account, eb EventBus.Bus) *ProxyManager {
	return &ProxyManager{
		clientWNMap:          sync.Map{},
		clientPeerMap:        sync.Map{},
		actorCtx:             actCtx,
		ctx:                  ctx,
		host:                 host,
		RegisterProxyTimeout: common.RegisterProxyTimeout,
		NewCircuitTimeout:    common.NewCircuitTimeout,
		DecryptReqTimeout:    common.DecryptReqTimeout,
		RetryTimes:           common.RetryTimes,
		circuitTask:          sync.Map{},
		Account:              acc,
		eb:                   eb,
	}
}

func (manager *ProxyManager) Start() {
	props := actor.PropsFromProducer(func() actor.Actor {
		return manager
	})
	manager.proxyPid = manager.actorCtx.Spawn(props)
}

func (manager *ProxyManager) Pid() *actor.PID {
	return manager.proxyPid
}

func (manager *ProxyManager) SetPid(relayPid *actor.PID, ackPid *actor.PID, gossipPid *actor.PID) {
	manager.relayPid = relayPid
	manager.ackPid = ackPid
	manager.gossipPid = gossipPid
}

func (manager *ProxyManager) AddClient(wnIdHash string, info ClientInfo) {
	manager.clientWNMap.Store(wnIdHash, info)
	manager.clientPeerMap.Store(info.PeerID.String(), info.WhiteNoiseID)
}

func (manager *ProxyManager) RemoveClient(peerIdString string) {
	v, ok := manager.clientPeerMap.Load(peerIdString)
	if !ok {
		manager.clientPeerMap.Delete(peerIdString)
		return
	}
	whiteNoiseID := v.(crypto.WhiteNoiseID)
	manager.clientWNMap.Delete(whiteNoiseID.Hash())
	manager.clientPeerMap.Delete(peerIdString)
}

func (manager *ProxyManager) GetClient(wnIdHash string) (ClientInfo, bool) {
	v, ok := manager.clientWNMap.Load(wnIdHash)
	if !ok {
		return ClientInfo{}, ok
	}
	return v.(ClientInfo), ok
}

func (manager *ProxyManager) AddNewCircuitTask(sessionId string, whitenoiseId crypto.WhiteNoiseID) {
	manager.circuitTask.Store(sessionId, whitenoiseId)
}

func (manager *ProxyManager) GetCircuitTask(sessionId string) (crypto.WhiteNoiseID, bool) {
	id, ok := manager.circuitTask.Load(sessionId)
	if !ok {
		return crypto.WhiteNoiseID{}, ok
	}
	return id.(crypto.WhiteNoiseID), ok
}

func (manager *ProxyManager) ProxyStreamHandler(stream network.Stream) {
	defer stream.Close()
	str := session.NewStream(stream, manager.ctx)
	payloadBytes, err := str.RW.ReadMsg()
	if err != nil {
		return
	}

	var request = pb.Request{}
	err = proto.Unmarshal(payloadBytes, &request)
	if err != nil {
		return
	}

	var ackMsg = pb.Ack{
		CommandId: request.ReqId,
		Result:    false,
		Data:      []byte{},
	}

	switch request.Reqtype {
	case pb.Reqtype_MainNetPeers:
		var getMainMetPeers = pb.MainNetPeers{}
		err = proto.Unmarshal(request.Data, &getMainMetPeers)
		if err != nil {
			ackMsg.Data = []byte("Unmarshal getMainMetPeers err")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		//request peers from dht network
		fut := manager.actorCtx.RequestFuture(manager.gossipPid, actorMsg.ReqDHTPeers{Max: int(getMainMetPeers.Max)}, common.RequestFutureDuration)
		res, err := fut.Result()
		if err != nil {
			ackMsg.Data = []byte("get dht peers err")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		resDHTPeers := res.(actorMsg.ResDHTPeers)
		peerInfos := make([]*pb.NodeInfo, len(resDHTPeers.PeerInfos))
		for i, _ := range peerInfos {
			var nodeInfo = pb.NodeInfo{}
			nodeInfo.Id = resDHTPeers.PeerInfos[i].ID.String()
			for _, addr := range resDHTPeers.PeerInfos[i].Addrs {
				nodeInfo.Addr = append(nodeInfo.Addr, addr.String())
			}
			peerInfos[i] = &nodeInfo
		}
		var peersList = pb.PeersList{Peers: peerInfos}
		data, err := proto.Marshal(&peersList)
		if err != nil {
			ackMsg.Data = []byte("marshall peersList err")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		ackMsg.Data = data
		ackMsg.Result = true
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{
			Ack:    &ackMsg,
			PeerId: str.RemotePeer,
		})

	case pb.Reqtype_NewProxy:
		var newProxyReq = pb.NewProxy{}
		err = proto.Unmarshal(request.Data, &newProxyReq)
		if err != nil {
			ackMsg.Data = []byte("Unmarshal newProxy err")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		whiteNoiseID, err := crypto.WhiteNoiseIDfromString(newProxyReq.WhiteNoiseID)
		if err != nil {
			log.Debug("WhiteNoiseIDfromString err:", err)
			ackMsg.Result = false
			ackMsg.Data = []byte(err.Error())
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		idHash := whiteNoiseID.Hash()
		if _, ok := manager.GetClient(idHash); ok {
			ackMsg.Data = []byte("Proxy already")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}

		duration, err := time.ParseDuration(newProxyReq.Time)
		if err != nil {
			duration = ProxySerivceTime
		}

		manager.AddClient(idHash, ClientInfo{
			WhiteNoiseID: whiteNoiseID,
			PeerID:       str.RemotePeer,
			state:        1,
			time:         duration,
		})

		ackMsg.Result = true
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
		log.Infof("Add new client %v", str.RemotePeer)
	case pb.Reqtype_NewCircuit:
		errMsg, err := manager.HandleNewCircuit(&request, str)
		if err != nil {
			log.Error("handle new circuit err", err)
			ackMsg.Data = errMsg
			ackMsg.Result = false
		} else {
			ackMsg.Result = true
		}
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
	case pb.Reqtype_DecryptGossip:
		plaintext, err := manager.HandleDecrypt(&request, str)
		if err != nil {
			ackMsg.Result = false
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{
				Ack:    &ackMsg,
				PeerId: str.RemotePeer,
			})
			break
		}
		ackMsg.Result = true
		ackMsg.Data = plaintext
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{
			Ack:    &ackMsg,
			PeerId: str.RemotePeer,
		})
	case pb.Reqtype_NegPlainText:
		log.Debug("handle encrypt neg")
		cypher, err := manager.HandleEncrypt(&request, str)
		if err != nil {
			log.Debug(err)
			ackMsg.Result = false
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{
				Ack:    &ackMsg,
				PeerId: str.RemotePeer,
			})
			break
		}
		ackMsg.Result = true
		ackMsg.Data = cypher
		manager.actorCtx.Request(manager.ackPid, ack.ReqAck{
			Ack:    &ackMsg,
			PeerId: str.RemotePeer,
		})
	case pb.Reqtype_UnRegisterType:
		log.Debug("handle unregister")
		manager.UnRegisterClient(str.RemotePeer)
	}
}

func (manager *ProxyManager) HandleDecrypt(request *pb.Request, str session.Stream) ([]byte, error) {
	var reqDecrypt pb.Decrypt
	err := proto.Unmarshal(request.Data, &reqDecrypt)
	if err != nil {
		return nil, err
	}
	//priv := manager.Account.GetECIESPrivKey()
	//plain, err := crypto.ECIESDecrypt(priv, reqDecrypt.Cypher)
	plain, err := manager.Account.GetPrivateKey().ECIESDecrypt(reqDecrypt.Cypher)
	if err != nil {
		return nil, err
	}
	var neg pb.Negotiate
	err = proto.Unmarshal(plain, &neg)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func (manager *ProxyManager) HandleEncrypt(request *pb.Request, str session.Stream) ([]byte, error) {
	var negPlaintext pb.NegPlaintext
	err := proto.Unmarshal(request.Data, &negPlaintext)
	if err != nil {
		return nil, err
	}
	whitenoise, ok := manager.GetCircuitTask(negPlaintext.SessionId)
	if !ok {
		return nil, errors.New("session not exist")
	}
	pk, err := whitenoise.PublicKey()
	if err != nil {
		return nil, err
	}
	negCypherData, err := pk.ECIESEncrypt(negPlaintext.Neg, cr.Reader)
	if err != nil {
		return nil, err
	}
	return negCypherData, nil
}

func (manager *ProxyManager) HandleNewCircuit(request *pb.Request, str session.Stream) ([]byte, error) {
	var newCircuit = pb.NewCircuit{}
	err := proto.Unmarshal(request.Data, &newCircuit)
	if err != nil {
		errMsg := []byte("Unmarshal newCircuit err")
		return errMsg, errors.New(string(errMsg))
	}

	if clientInfo, ok := manager.GetClient(newCircuit.From); !ok || clientInfo.PeerID != str.RemotePeer {
		errMsg := []byte("Unmarshal newCircuit err")
		return errMsg, errors.New(string(errMsg))
	}

	fut := manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqGetSession{Id: newCircuit.SessionId}, common.RequestFutureDuration)
	res, err := fut.Result()
	if err != nil {
		return []byte{}, err
	}
	sess := res.(relay.ResGetSession).Session
	ok := res.(relay.ResGetSession).Ok

	if !ok {
		errMsg := []byte("No stream for this session yet")
		return errMsg, errors.New(string(errMsg))
	} else if sess.IsReady() {
		errMsg := []byte("Session is full")
		return errMsg, errors.New(string(errMsg))
	}

	//server and client connect to the same proxy
	if clientInfo, ok := manager.GetClient(newCircuit.To); ok {
		log.Info("client server both to me")
		//new session to the answer role
		fut := manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqNewSessiontoPeer{
			PeerID:    clientInfo.PeerID,
			SessionID: newCircuit.SessionId,
			MyRole:    common.ExitRole,
			OtherRole: common.AnswerRole,
		}, common.RequestFutureDuration)

		res, err := fut.Result()
		if err != nil {
			return nil, err
		}
		resErr := res.(relay.ResError).Err
		if resErr != nil {
			manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: newCircuit.SessionId})
			return nil, resErr
		}

		//send build circuit success to clients (act like a joint node)
		log.Debug("send circuit success signal")
		relayMsg := relay.NewCircuitSuccess(newCircuit.SessionId)
		fut = manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqSendRelay{
			SessionId: newCircuit.SessionId,
			Data:      relayMsg,
		}, common.RequestFutureDuration)

		res, err = fut.Result()
		if err != nil {
			return nil, err
		}
		resErr = res.(relay.ResError).Err
		if resErr != nil {
			return nil, resErr
		}
		return nil, nil
	}

	invalid := make(map[string]bool)
	tryJoinSuccess := false
	var join = core.PeerID("")
	source := rand.NewSource(time.Now().UnixNano())
	fut = manager.actorCtx.RequestFuture(manager.gossipPid, actorMsg.ReqDHTPeers{Max: common.ReqDHTPeersMaxAmount}, common.RequestFutureDuration)
	res, err = fut.Result()
	if err != nil {
		return nil, err
	}
	resDHTPeers := res.(actorMsg.ResDHTPeers)
	peers := resDHTPeers.PeerInfos
	for i := 0; i < manager.RetryTimes; i++ {
		startIndex := rand.New(source).Int()
		for j := 0; j < len(peers); j++ {
			startIndex++
			index := startIndex % len(peers)
			id := peers[index].ID
			if _, ok := invalid[id.String()]; id != manager.host.ID() && id != str.RemotePeer && !ok {
				hash := sha256.Sum256([]byte(id.String()))
				if secure.EncodeMSGIDHash(hash[:]) != newCircuit.To {
					join = id
					break
				}
			}
		}
		fut = manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqNewSessiontoPeer{
			PeerID:    join,
			SessionID: newCircuit.SessionId,
			MyRole:    common.EntryRole,
			OtherRole: common.JointRole,
		}, common.RequestFutureDuration)
		res, err := fut.Result()
		if err != nil {
			return nil, err
		}
		resErr := res.(relay.ResError).Err
		if resErr != nil {
			invalid[join.String()] = true
		} else {
			tryJoinSuccess = true
			break
		}
	}

	if !tryJoinSuccess {
		log.Warnf("cannot find joint node %v", newCircuit.SessionId)
		manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: newCircuit.SessionId})
		errMsg := []byte("Cannot find joint node " + err.Error())
		return errMsg, errors.New(string(errMsg))
	}
	//request caller to encrypt gossip msg
	var neg = pb.Negotiate{
		Join:        join.String(),
		SessionId:   newCircuit.SessionId,
		Destination: newCircuit.To,
		Sig:         []byte{},
	}
	negData, _ := proto.Marshal(&neg)
	negCypher, err := manager.EncryptGossip(negData, str.RemotePeer, newCircuit.SessionId)
	if err != nil {
		return nil, err
	}

	log.Infof("Gossip for session %v, joint node %v", newCircuit.SessionId, join.String())
	fut = manager.actorCtx.RequestFuture(manager.gossipPid, actorMsg.ReqGossipJoint{
		DesHash:   newCircuit.To,
		NegCypher: negCypher,
	}, common.RequestFutureDuration)
	res, err = fut.Result()
	if err != nil {
		return nil, err
	}
	resErr := res.(actorMsg.ResError).Err
	if resErr != nil {
		log.Warnf("Gossip err %v", resErr)
		manager.actorCtx.Request(manager.relayPid, relay.ReqCloseCircuit{SessionId: newCircuit.SessionId})
		errMsg := []byte("GossipJoint err" + resErr.Error())
		return errMsg, err
	}
	log.Debug("Sending probe signal")
	//send probe signal to joint node
	probeSignal, _ := relay.NewProbeSignal(newCircuit.SessionId)
	fut = manager.actorCtx.RequestFuture(manager.relayPid, relay.ReqSendRelay{
		SessionId: newCircuit.SessionId,
		Data:      probeSignal,
	}, common.RequestFutureDuration)
	res, err = fut.Result()
	if err != nil {
		return nil, err
	}
	resErr = res.(relay.ResError).Err
	if resErr != nil {
		return nil, resErr
	}
	return nil, nil
}

func (manager *ProxyManager) DecryptGossip(des core.PeerID, cypher []byte) (string, string, error) {
	streamRaw, err := manager.NewProxyStream(des)
	if err != nil {
		return "", "", err
	}

	s := session.NewStream(streamRaw, manager.ctx)

	req := pb.Request{
		ReqId:   "",
		From:    manager.host.ID().String(),
		Reqtype: pb.Reqtype_DecryptGossip,
		Data:    []byte{},
	}

	decReq := pb.Decrypt{
		Destination: des.String(),
		Cypher:      cypher,
	}

	data, _ := proto.Marshal(&decReq)
	req.Data = data
	reqNoIDBytes, _ := proto.Marshal(&req)
	hash := sha256.Sum256(reqNoIDBytes)
	reqID := secure.EncodeMSGIDHash(hash[:])
	req.ReqId = reqID

	pl, _ := proto.Marshal(&req)
	err = s.RW.WriteMsg(pl)
	if err != nil {
		return "", "", err
	}

	//wait for plaintext ack
	task := ack.Task{
		Id:      req.ReqId,
		Channel: make(chan ack.Result),
	}
	manager.actorCtx.Request(manager.ackPid, ack.ReqAddTask{T: task})
	defer manager.actorCtx.Request(manager.ackPid, ack.ReqDeleteTask{Id: task.Id})
	timeout := time.After(manager.DecryptReqTimeout)
	select {
	case <-timeout:
		return "", "", errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			return "", "", errors.New("client decrypt err")
		}
		var neg pb.Negotiate
		err := proto.Unmarshal(result.Data, &neg)
		if err != nil {
			return "", "", err
		}
		return neg.SessionId, neg.Join, nil
	}
}

func (manager *ProxyManager) EncryptGossip(plaintext []byte, caller peer.ID, sessionID string) ([]byte, error) {
	streamRaw, err := manager.NewProxyStream(caller)
	if err != nil {
		return nil, err
	}
	s := session.NewStream(streamRaw, manager.ctx)

	req := pb.Request{
		ReqId:   "",
		From:    manager.host.ID().String(),
		Reqtype: pb.Reqtype_NegPlainText,
		Data:    []byte{},
	}
	negPlain := pb.NegPlaintext{
		SessionId: sessionID,
		Neg:       plaintext,
	}
	data, _ := proto.Marshal(&negPlain)
	req.Data = data
	reqNoIDBytes, _ := proto.Marshal(&req)
	hash := sha256.Sum256(reqNoIDBytes)
	reqID := secure.EncodeMSGIDHash(hash[:])
	req.ReqId = reqID

	pl, _ := proto.Marshal(&req)
	err = s.RW.WriteMsg(pl)
	if err != nil {
		return nil, err
	}

	//wait for cyphertext ack
	task := ack.Task{
		Id:      req.ReqId,
		Channel: make(chan ack.Result),
	}
	manager.actorCtx.Request(manager.ackPid, ack.ReqAddTask{T: task})
	defer manager.actorCtx.Request(manager.ackPid, ack.ReqDeleteTask{Id: task.Id})
	timeout := time.After(manager.DecryptReqTimeout)
	select {
	case <-timeout:
		return nil, errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			return nil, errors.New("caller encrypt gossip msg err")
		}
		return result.Data, nil
	}
}

func (manager *ProxyManager) NewProxyStream(id core.PeerID) (network.Stream, error) {
	return manager.host.NewStream(manager.ctx, id, protocol.ID(PROXY_PROTOCOL))
}

//todo:close client's existing circuits
func (manager *ProxyManager) UnRegisterClient(id core.PeerID) {
	manager.RemoveClient(id.String())
}
