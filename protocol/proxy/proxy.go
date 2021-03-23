package proxy

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"math/rand"
	"sync"
	"time"
	"whitenoise/common"
	"whitenoise/common/log"
	"whitenoise/internal/actorMsg"
	"whitenoise/internal/pb"
	"whitenoise/network/session"
	"whitenoise/protocol/ack"
	"whitenoise/protocol/relay"
	"whitenoise/secure"
)

const PROXY_PROTOCOL string = "proxyprotocol"
const ProxySerivceTime time.Duration = time.Hour

type ProxyManager struct {
	mut                  sync.Mutex
	clientMap            map[string]ClientInfo
	actorCtx             *actor.RootContext
	ctx                  context.Context
	ackPid               *actor.PID
	host                 core.Host
	gossipPid            *actor.PID
	relayPid             *actor.PID
	proxyPid             *actor.PID
	RegisterProxyTimeout time.Duration
	NewCircuitTimeout    time.Duration
	RetryTimes           int
}

type ClientInfo struct {
	PeerID core.PeerID
	state  int
	time   time.Duration
}

func NewProxyService(host core.Host, ctx context.Context, actCtx *actor.RootContext) *ProxyManager {
	return &ProxyManager{
		mut:                  sync.Mutex{},
		clientMap:            make(map[string]ClientInfo),
		actorCtx:             actCtx,
		ctx:                  ctx,
		host:                 host,
		RegisterProxyTimeout: common.RegisterProxyTimeout,
		NewCircuitTimeout:    common.NewCircuitTimeout,
		RetryTimes:           common.RetryTimes,
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

func (manager *ProxyManager) AddClient(id string, info ClientInfo) {
	//defer man.mut.Unlock()
	//man.mut.Lock()
	manager.clientMap[id] = info
}

func (manager *ProxyManager) RemoveClient(id string) {
	//defer service.mut.Unlock()
	//service.mut.Lock()
	delete(manager.clientMap, id)
}

func (manager *ProxyManager) GetClient(id string) (ClientInfo, bool) {
	//defer service.mut.Unlock()
	//service.mut.Lock()
	info, ok := manager.clientMap[id]
	return info, ok
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
	case pb.Reqtype_GetOnlineNodes:
	case pb.Reqtype_NewProxy:
		var newProxyReq = pb.NewProxy{}
		err = proto.Unmarshal(request.Data, &newProxyReq)
		if err != nil {
			ackMsg.Data = []byte("Unmarshal newProxy err")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}
		if _, ok := manager.GetClient(newProxyReq.IdHash); ok {
			ackMsg.Data = []byte("Proxy already")
			manager.actorCtx.Request(manager.ackPid, ack.ReqAck{Ack: &ackMsg, PeerId: str.RemotePeer})
			break
		}

		duration, err := time.ParseDuration(newProxyReq.Time)
		if err != nil {
			duration = ProxySerivceTime
		}
		manager.AddClient(newProxyReq.IdHash, ClientInfo{
			PeerID: str.RemotePeer,
			state:  1,
			time:   duration,
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
	}
}

func (manager *ProxyManager) HandleNewCircuit(request *pb.Request, str session.Stream) ([]byte, error) {
	var newCircuit = pb.NewCircuit{}
	err := proto.Unmarshal(request.Data, &newCircuit)
	if err != nil {
		errMsg := []byte("Unmarshal newCircuit err")
		return errMsg, errors.New(string(errMsg))
	}

	hash := sha256.Sum256([]byte(str.RemotePeer.String()))
	if _, ok := manager.GetClient(secure.EncodeID(hash[:])); !ok {
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
	//todo: this scheme needs review
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
		//relayMsg := relay.NewRelayMsg([]byte("build circuit success"), newCircuit.SessionId)
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
	peers := manager.host.Peerstore().Peers()
	for i := 0; i < manager.RetryTimes; i++ {
		//todo:break this loop when not enough node in the mainnet (need review)
		startIndex := rand.New(source).Int()
		for j := 0; j < len(peers); j++ {
			startIndex++
			index := startIndex % len(peers)
			id := peers[index]
			if _, ok := invalid[id.String()]; id != manager.host.ID() && id != str.RemotePeer && !ok {
				hash := sha256.Sum256([]byte(id.String()))
				if secure.EncodeID(hash[:]) != newCircuit.To {
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

	log.Infof("Gossip for session %v, joint node %v", newCircuit.SessionId, join.String())
	fut = manager.actorCtx.RequestFuture(manager.gossipPid, actorMsg.ReqGossipJoint{
		DesHash:   newCircuit.To,
		Joint:     join,
		SessionId: newCircuit.SessionId,
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

	//send probe signal to joint node
	probeSignal, _ := relay.NewProbeSignal(newCircuit.SessionId)
	//todo:wait for result and handle err
	manager.actorCtx.Request(manager.relayPid, relay.ReqSendRelay{
		SessionId: newCircuit.SessionId,
		Data:      probeSignal,
	})
	return nil, nil
}
