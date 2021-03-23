package noise

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"time"
	"whitenoise/common"
	"whitenoise/common/config"
	"whitenoise/common/log"
	"whitenoise/internal/pb"
	"whitenoise/network/session"
	"whitenoise/protocol/ack"
	"whitenoise/protocol/command"
	"whitenoise/protocol/proxy"
	"whitenoise/protocol/relay"
	"whitenoise/secure"
)

type NoiseService struct {
	host         host.Host
	ctx          context.Context
	actCtx       *actor.RootContext
	ackManager   *ack.AckManager
	proxyManager *proxy.ProxyManager
	relayManager *relay.RelayMsgManager
	cmdManager   *command.CmdManager
	ProxyNode    core.PeerID
	Role         config.ServiceMode
}

func (service *NoiseService) Host() host.Host {
	return service.host
}

func (service *NoiseService) Ctx() context.Context {
	return service.ctx
}

func (service *NoiseService) Relay() *relay.RelayMsgManager {
	return service.relayManager
}

func (service *NoiseService) Cmd() *command.CmdManager {
	return service.cmdManager
}

func (service *NoiseService) TryConnect(peer core.PeerAddrInfo) {
	err := service.host.Connect(service.ctx, peer)
	if err != nil {
		log.Errorf("connect to %v error: %v", peer.ID, err)
	} else {
	}
}

func NewNoiseService(ctx context.Context, actCtx *actor.RootContext, cfg *config.NetworkConfig, h core.Host, key crypto.PrivKey) (*NoiseService, error) {
	service := NoiseService{
		host:         h,
		ctx:          ctx,
		actCtx:       actCtx,
		ackManager:   ack.NewAckManager(h, ctx, actCtx),
		proxyManager: proxy.NewProxyService(h, ctx, actCtx),
		relayManager: relay.NewRelayMsgManager(h, ctx, actCtx, cfg.Mode, key),
		cmdManager:   command.NewCmdHandler(h, ctx, actCtx),
		Role:         cfg.Mode,
	}

	service.host.SetStreamHandler(protocol.ID(relay.RelayProtocol), service.relayManager.RelayStreamHandler)
	service.host.SetStreamHandler(protocol.ID(command.CMD_PROTOCOL), service.cmdManager.CmdStreamHandler)
	service.host.SetStreamHandler(protocol.ID(ack.ACK_PROTOCOL), service.ackManager.AckStreamHandler)
	service.host.SetStreamHandler(protocol.ID(proxy.PROXY_PROTOCOL), service.proxyManager.ProxyStreamHandler)

	return &service, nil
}

func (service *NoiseService) AckPid() *actor.PID {
	return service.ackManager.Pid()
}

func (service *NoiseService) ProxyPid() *actor.PID {
	return service.proxyManager.Pid()
}

func (service *NoiseService) RelayPid() *actor.PID {
	return service.relayManager.Pid()
}

func (service *NoiseService) CmdPid() *actor.PID {
	return service.cmdManager.Pid()
}

func (service *NoiseService) Start() {
	log.Infof("start service\n PeerId: %v\n ", peer.Encode(service.host.ID()))
	log.Info("MultiAddrs:")
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", service.host.ID().Pretty()))
	for _, addr := range service.host.Addrs() {
		log.Infof("%v \n", addr.Encapsulate(hostAddr))
	}

	service.ackManager.Start()
	service.proxyManager.Start()
	service.relayManager.Start()
	service.cmdManager.Start()
}

func (service *NoiseService) SetPid(gossipPid *actor.PID) {
	service.proxyManager.SetPid(service.RelayPid(), service.AckPid(), gossipPid)
	service.relayManager.SetPid(service.AckPid())
	service.cmdManager.SetPid(service.RelayPid(), service.AckPid())
}

func (service *NoiseService) RegisterProxy(proxyId core.PeerID) error {
	log.Info(proxyId.String())
	streamRaw, err := service.host.NewStream(service.ctx, proxyId, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := session.NewStream(streamRaw, service.ctx)
	hash := sha256.Sum256([]byte(service.host.ID().String()))
	newProxy := pb.NewProxy{
		Time:   time.Hour.String(),
		IdHash: secure.EncodeID(hash[:]),
	}
	data, err := proto.Marshal(&newProxy)
	if err != nil {
		return err
	}
	request := pb.Request{
		ReqId:   "",
		From:    service.host.ID().String(),
		Reqtype: pb.Reqtype_NewProxy,
		Data:    data,
	}
	noID, err := proto.Marshal(&request)
	if err != nil {
		return err
	}
	hash = sha256.Sum256(noID)
	request.ReqId = secure.EncodeID(hash[:])
	reqData, err := proto.Marshal(&request)

	err = stream.RW.WriteMsg(reqData)
	if err != nil {
		return err
	}

	task := ack.Task{
		Id:      request.ReqId,
		Channel: make(chan ack.Result),
	}
	service.ackManager.AddTask(task)
	defer service.ackManager.DeletTask(request.ReqId)
	select {
	case <-time.After(service.proxyManager.RegisterProxyTimeout):
		return errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			return errors.New("register rejected: " + string(result.Data))
		}
	}
	service.ProxyNode = proxyId
	log.Infof("Register to proxy %v", proxyId)
	return nil
}

func (service *NoiseService) NewCircuit(des core.PeerID, sessionId string) (err error) {
	defer func() {
		if err != nil {
			log.Error(err)
			service.relayManager.CloseCircuit(sessionId)
		}
	}()
	if _, ok := service.relayManager.GetSession(sessionId); ok {
		return errors.New("circuit with same sessionId already exist")
	}

	err = service.relayManager.NewSessionToPeer(service.ProxyNode, sessionId, common.CallerRole, common.EntryRole)
	if err != nil {
		return err
	}
	log.Debug("new session to peer finished")

	streamRaw, err := service.host.NewStream(service.ctx, service.ProxyNode, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := session.NewStream(streamRaw, service.ctx)

	//Add new circuitConn for this session in MsgManager
	service.relayManager.AddCircuitConnCaller(sessionId, des)

	hashID := sha256.Sum256([]byte(des.String()))
	newCircuit := pb.NewCircuit{
		To:        secure.EncodeID(hashID[:]),
		SessionId: sessionId,
	}

	data, err := proto.Marshal(&newCircuit)
	if err != nil {
		return err
	}

	request := pb.Request{
		ReqId:   "",
		From:    service.host.ID().String(),
		Reqtype: pb.Reqtype_NewCircuit,
		Data:    data,
	}
	noId, err := proto.Marshal(&request)
	hash := sha256.Sum256(noId[:])
	request.ReqId = secure.EncodeID(hash[:])
	reqData, err := proto.Marshal(&request)

	err = stream.RW.WriteMsg(reqData)
	if err != nil {
		return err
	}

	task := ack.Task{
		Id:      request.ReqId,
		Channel: make(chan ack.Result),
	}
	service.ackManager.AddTask(task)
	defer service.ackManager.DeletTask(request.ReqId)

	select {
	case <-time.After(service.proxyManager.NewCircuitTimeout):
		return errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			service.relayManager.CloseCircuit(sessionId)
			return errors.New("new circuit rejected: " + string(result.Data))
		}
	}
	return nil
}
