package gossip

import (
	"context"
	"crypto/sha256"
	"github.com/AsynkronIT/protoactor-go/actor"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/mr-tron/base58"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"time"
	"github.com/Evanesco-Labs/WhiteNoise/common"
	"github.com/Evanesco-Labs/WhiteNoise/common/config"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/command"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/proxy"
	"github.com/Evanesco-Labs/WhiteNoise/protocol/relay"
)

const NoiseTopic string = "noise_topic"

type DHTService struct {
	actorCtx   *actor.RootContext
	ctx        context.Context
	ps         *pubsub.PubSub
	noiseTopic *pubsub.Topic
	noiseSub   *pubsub.Subscription
	dht        *kaddht.IpfsDHT
	proxyPid   *actor.PID
	relayPid   *actor.PID
	cmdPid     *actor.PID
	gossipPid  *actor.PID
	host       host.Host
	RetryTimes int
}

func (service *DHTService) Dht() *kaddht.IpfsDHT {
	return service.dht
}

func NewDHTService(ctx context.Context, actCtx *actor.RootContext, cfg *config.NetworkConfig, host core.Host, dht *kaddht.IpfsDHT) (*DHTService, error) {
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithNoAuthor(), pubsub.WithMessageIdFn(MessageID))
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return nil, err
	}
	noiseTopic, err := ps.Join(NoiseTopic)
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return nil, err
	}

	noiseSub, err := noiseTopic.Subscribe()
	if err != nil {
		log.Error("NewPubsubService err: ", err)
		return nil, err
	}

	pubsubService := &DHTService{
		actorCtx:   actCtx,
		ctx:        ctx,
		ps:         ps,
		noiseTopic: noiseTopic,
		noiseSub:   noiseSub,
		dht:        dht,
		host:       host,
		RetryTimes: common.RetryTimes,
	}

	return pubsubService, nil
}

func (service *DHTService) Start(cfg *config.NetworkConfig) {
	err := service.InitDHT([]string{cfg.BootStrapPeers})
	if err != nil {
		panic(err)
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return service
	})
	if cfg.Mode == config.ServerMode {
		go service.handleNoiseMsg(service.ctx)
	}
	service.gossipPid = service.actorCtx.Spawn(props)
}

func (service *DHTService) Pid() *actor.PID {
	return service.gossipPid
}

func (service *DHTService) SetPid(proxyPid, relayPid, cmdPid *actor.PID) {
	service.proxyPid = proxyPid
	service.relayPid = relayPid
	service.cmdPid = cmdPid
}

func (service *DHTService) InitDHT(bootStrapPeers []string) error {
	c := context.Background()
	err := service.dht.Bootstrap(c)
	if err != nil {
		return err
	}
	service.dht.RefreshRoutingTable()
	return nil
}

func (service *DHTService) NoisePublish(data []byte) error {
	return service.noiseTopic.Publish(service.ctx, data)
}

func (service *DHTService) GetDHTPeers(max int) []peer.AddrInfo {
	peerIDs := service.dht.RoutingTable().ListPeers()
	if len(peerIDs) > max {
		peerIDs = peerIDs[:max]
	}
	peerInfos := make([]peer.AddrInfo, 0)
	for _, id := range peerIDs {
		info := service.dht.FindLocal(id)
		peerInfos = append(peerInfos, info)
	}
	return peerInfos
}

func (service *DHTService) handleNoiseMsg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			break
		}
		msg, err := service.noiseSub.Next(service.ctx)
		if err != nil {
			log.Errorf("noise sub err %v", err)
			return
		}
		var neg pb.EncryptedNeg
		err = proto.Unmarshal(msg.Data, &neg)
		if err != nil {
			log.Errorf("Unmarshall gossip error: %v", err)
			continue
		}
		fut := service.actorCtx.RequestFuture(service.proxyPid, proxy.ReqGetClient{Destination: neg.Des}, common.RequestFutureDuration)
		res, err := fut.Result()
		clientInfo := res.(proxy.ResGetClient).Info
		ok := res.(proxy.ResGetClient).Ok
		if ok {
			log.Debugf("Handling gossip for my client %v", clientInfo.PeerID.String())
			go service.handleGossipMsg(clientInfo, &neg)
		}
	}
}

func (service *DHTService) handleGossipMsg(clientInfo proxy.ClientInfo, negEnc *pb.EncryptedNeg) {
	fut := service.actorCtx.RequestFuture(service.proxyPid, proxy.ReqDecrypt{
		CipherText: negEnc.Cypher,
		Des:        clientInfo.PeerID,
	}, common.RequestFutureDuration)
	res, err := fut.Result()
	if err != nil {
		log.Error(err)
		return
	}
	neg := res.(proxy.ResDecrypt)
	if neg.Err != nil {
		log.Error(neg.Err)
		return
	}

	//new session to answer role
	fut = service.actorCtx.RequestFuture(service.relayPid, relay.ReqNewSessiontoPeer{
		PeerID:    clientInfo.PeerID,
		SessionID: neg.SessionId,
		MyRole:    common.ExitRole,
		OtherRole: common.AnswerRole,
	}, common.RequestFutureDuration)

	res, err = fut.Result()
	if err != nil {
		log.Error(err)
		return
	}
	resErr := res.(relay.ResError).Err
	if resErr != nil {
		log.Errorf("New session to destination err:%v", resErr)
		service.actorCtx.Request(service.relayPid, relay.ReqCloseCircuit{SessionId: neg.SessionId})
		return
	}

	joinNode, err := peer.Decode(neg.Join)
	var relayId core.PeerID

	if joinNode == service.host.ID() {
		log.Info("act as both joint and exit")
		log.Debug("send circuit success signal")
		msg := relay.NewCircuitSuccess(neg.SessionId)
		fut = service.actorCtx.RequestFuture(service.relayPid, relay.ReqSendRelay{
			SessionId: neg.SessionId,
			Data:      msg,
		}, common.RequestFutureDuration)
		res, err = fut.Result()
		if err != nil {
			log.Error(err)
			return
		}
		resErr = res.(relay.ResError).Err
		if resErr != nil {
			log.Error(resErr)
		}
		return
	}

	invalid := make(map[string]bool)
	tryRelaySuccess := false
	source := rand.NewSource(time.Now().UnixNano())
	peers := service.GetDHTPeers(common.ReqDHTPeersMaxAmount)

	for i := 0; i < service.RetryTimes; i++ {
		startIndex := rand.New(source).Int()
		for j := 0; j < len(peers); j++ {
			startIndex++
			index := startIndex % len(peers)
			id := peers[index].ID
			if _, ok := invalid[id.String()]; !ok && id != joinNode && id != service.host.ID() {
				relayId = id
				break
			}
		}
		//try set new session to relay
		fut = service.actorCtx.RequestFuture(service.relayPid, relay.ReqNewSessiontoPeer{
			PeerID:    relayId,
			SessionID: neg.SessionId,
			MyRole:    common.ExitRole,
			OtherRole: common.RelayRole,
		}, common.RequestFutureDuration)
		res, err = fut.Result()
		if err != nil {
			log.Error(err)
			return
		}
		resErr = res.(relay.ResError).Err
		if resErr != nil {
			invalid[relayId.String()] = true
			continue
		} else {
			tryRelaySuccess = true
			break
		}
	}

	if !tryRelaySuccess {
		log.Errorf("No valid node for relay role err %v", err)
		service.actorCtx.Request(service.relayPid, relay.ReqCloseCircuit{SessionId: neg.SessionId})
		return
	}
	log.Debugf("Chose relay node %v", relayId)
	//expend relay node to joint node
	fut = service.actorCtx.RequestFuture(service.cmdPid, command.ReqExpendSession{
		Relay:     relayId,
		Joint:     joinNode,
		SessionId: neg.SessionId,
	}, common.RequestFutureDuration)
	res, err = fut.Result()
	if err != nil {
		log.Error(err)
		return
	}
	resErr = res.(command.ResError).Err
	if resErr != nil {
		log.Errorf("Expend session err %v", resErr)
		service.actorCtx.Request(service.relayPid, relay.ReqCloseCircuit{SessionId: neg.SessionId})
		return
	}

	log.Infof("set relay node %v for session %v", relayId.String(), neg.SessionId)

	//send probe signal to joint node
	relayData, err := relay.NewProbeSignal(neg.SessionId)
	if err != nil {
		log.Errorf("NewProbeSignal err %v", err)
	}

	fut = service.actorCtx.RequestFuture(service.relayPid, relay.ReqSendRelay{
		SessionId: neg.SessionId,
		Data:      relayData,
	}, common.RequestFutureDuration)
	res, err = fut.Result()
	if err != nil {
		log.Error(err)
		return
	}
	resErr = res.(relay.ResError).Err
	if resErr != nil {
		log.Errorf("NewProbeSignal err %v", resErr)
	}
}

func MessageID(pmsg *pubsub_pb.Message) string {
	hash := sha256.Sum256([]byte(pmsg.String()))
	return base58.Encode(hash[:])
}
