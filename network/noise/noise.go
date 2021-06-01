package noise

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/asaskevich/EventBus"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"time"
	"whitenoise/common"
	"whitenoise/common/account"
	"whitenoise/common/config"
	"whitenoise/common/log"
	crypto2 "whitenoise/crypto"
	"whitenoise/internal/pb"
	"whitenoise/network/session"
	"whitenoise/protocol/ack"
	"whitenoise/protocol/command"
	"whitenoise/protocol/proxy"
	"whitenoise/protocol/relay"
	"whitenoise/secure"
)

type NoiseService struct {
	host          host.Host
	ctx           context.Context
	actCtx        *actor.RootContext
	ackManager    *ack.AckManager
	proxyManager  *proxy.ProxyManager
	relayManager  *relay.RelayMsgManager
	cmdManager    *command.CmdManager
	ProxyNode     core.PeerID
	Role          config.ServiceMode
	Account       *account.Account
	eventBus      EventBus.Bus
	BootstrapPeer peer.ID
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

func NewNoiseService(ctx context.Context, actCtx *actor.RootContext, cfg *config.NetworkConfig, h core.Host, key crypto.PrivKey, acc *account.Account) (*NoiseService, error) {
	eb := EventBus.New()
	service := NoiseService{
		host:         h,
		ctx:          ctx,
		actCtx:       actCtx,
		ackManager:   ack.NewAckManager(h, ctx, actCtx, eb),
		proxyManager: proxy.NewProxyService(h, ctx, actCtx, acc, eb),
		relayManager: relay.NewRelayMsgManager(h, ctx, actCtx, cfg.Mode, key, acc, eb),
		cmdManager:   command.NewCmdHandler(h, ctx, actCtx, eb),
		Role:         cfg.Mode,
		Account:      acc,
		eventBus:     eb,
	}

	service.host.SetStreamHandler(protocol.ID(ack.ACK_PROTOCOL), service.ackManager.AckStreamHandler)
	service.host.SetStreamHandler(protocol.ID(proxy.PROXY_PROTOCOL), service.proxyManager.ProxyStreamHandler)

	if cfg.Mode != config.BootMode {
		service.host.SetStreamHandler(protocol.ID(relay.RelayProtocol), service.relayManager.RelayStreamHandler)
		service.host.SetStreamHandler(protocol.ID(command.CMD_PROTOCOL), service.cmdManager.CmdStreamHandler)
	}

	if cfg.BootStrapPeers == "" {
		return &service, nil
	}

	peerAddr := cfg.BootStrapPeers
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		log.Errorf("Parse bootstrap node address err %v", err)
		return nil, err
	}
	peerinfo, _ := peer.AddrInfoFromP2pAddr(maddr)
	service.host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)
	service.BootstrapPeer = peerinfo.ID
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

func (service *NoiseService) EventBus() EventBus.Bus {
	return service.eventBus
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

func (service *NoiseService) SetNotify(h host.Host, cfg *config.NetworkConfig) {
	notifiee := NoiseNotifiee{
		host:          h,
		actCtx:        service.actCtx,
		proxyPid:      service.ProxyPid(),
		relayPid:      service.RelayPid(),
		whiteListMode: cfg.WhiteList,
	}
	service.Host().Network().Notify(notifiee)
}

func (service *NoiseService) RegisterProxy(proxyId core.PeerID) error {
	streamRaw, err := service.host.NewStream(service.ctx, proxyId, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := session.NewStream(streamRaw, service.ctx)
	newProxy := pb.NewProxy{
		Time:         time.Hour.String(),
		WhiteNoiseID: service.Account.GetPublicKey().GetWhiteNoiseID().String(),
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
	hash := sha256.Sum256(noID)
	request.ReqId = secure.EncodeMSGIDHash(hash[:])
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

func (service *NoiseService) UnRegister() error {
	defer func() {
		service.ProxyNode = ""
	}()
	if service.ProxyNode == "" {
		return errors.New("no proxy yet")
	}
	streamRaw, err := service.host.NewStream(service.ctx, service.ProxyNode, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := session.NewStream(streamRaw, service.ctx)

	unReg := pb.UnRegister{
		SessionID: service.relayManager.GetSessionIDList(),
	}

	data, err := proto.Marshal(&unReg)
	if err != nil {
		return err
	}

	request := pb.Request{
		ReqId:   "",
		From:    service.Host().ID().String(),
		Reqtype: pb.Reqtype_UnRegisterType,
		Data:    data,
	}

	payloadNoID, err := proto.Marshal(&request)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(payloadNoID)

	request.ReqId = secure.EncodeMSGIDHash(hash[:])
	payload, err := proto.Marshal(&request)
	return stream.RW.WriteMsg(payload)
}

func (service *NoiseService) NewCircuit(remoteIDString string, sessionId string) (err error) {
	defer func() {
		if err != nil {
			log.Error(err)
			service.relayManager.CloseCircuit(sessionId)
		}
	}()

	desWhiteNoiseID, err := crypto2.WhiteNoiseIDfromString(remoteIDString)
	if err != nil {
		return err
	}

	service.proxyManager.AddNewCircuitTask(sessionId, desWhiteNoiseID)

	if _, ok := service.relayManager.GetSession(sessionId); ok {
		return errors.New("circuit with same sessionId already exist")
	}

	err = service.relayManager.NewSessionToPeer(service.ProxyNode, sessionId, common.CallerRole, common.EntryRole)
	if err != nil {
		return err
	}

	streamRaw, err := service.host.NewStream(service.ctx, service.ProxyNode, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := session.NewStream(streamRaw, service.ctx)

	//Add new circuitConn for this session in MsgManager
	service.relayManager.AddCircuitConnCaller(sessionId, desWhiteNoiseID)

	newCircuit := pb.NewCircuit{
		From:      service.Account.GetPublicKey().GetWhiteNoiseID().Hash(),
		To:        desWhiteNoiseID.Hash(),
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
	request.ReqId = secure.EncodeMSGIDHash(hash[:])
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
	timeout := time.After(service.proxyManager.NewCircuitTimeout)
	select {
	case <-timeout:
		return errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			service.relayManager.CloseCircuit(sessionId)
			return errors.New("new circuit rejected: " + string(result.Data))
		}
		return nil
	}
}

func (service *NoiseService) GetMainnetPeers(max int) ([]peer.AddrInfo, error) {
	streamRaw, err := service.host.NewStream(service.ctx, service.BootstrapPeer, protocol.ID(proxy.PROXY_PROTOCOL))
	if err != nil {
		return nil, err
	}
	stream := session.NewStream(streamRaw, service.ctx)
	getMainnetPeers := pb.MainNetPeers{Max: int32(max)}
	data, err := proto.Marshal(&getMainnetPeers)
	if err != nil {
		return nil, err
	}
	request := pb.Request{
		ReqId:   "",
		From:    "",
		Reqtype: pb.Reqtype_MainNetPeers,
		Data:    data,
	}
	reqNoId, err := proto.Marshal(&request)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(reqNoId)
	request.ReqId = secure.EncodeMSGIDHash(hash[:])
	reqData, err := proto.Marshal(&request)
	if err != nil {
		return nil, err
	}
	err = stream.RW.WriteMsg(reqData)
	if err != nil {
		return nil, err
	}

	task := ack.Task{
		Id:      request.ReqId,
		Channel: make(chan ack.Result),
	}
	service.ackManager.AddTask(task)
	defer service.ackManager.DeletTask(request.ReqId)
	timeout := time.After(common.GetMainnetPeersTimeout)
	select {
	case <-timeout:
		return nil, errors.New("timeout")
	case result := <-task.Channel:
		if !result.Ok {
			return nil, errors.New("get mainnet peers err")
		}
		var peerList pb.PeersList
		err := proto.Unmarshal(result.Data, &peerList)
		if err != nil {
			return nil, errors.New("unmarshall peerList err" + err.Error())
		}
		peerInfos := make([]peer.AddrInfo, 0)
		for _, nodeInfo := range peerList.Peers {
			peerID, err := peer.Decode(nodeInfo.Id)
			if err != nil {
				continue
			}
			peerinfo := peer.AddrInfo{
				ID:    peerID,
				Addrs: make([]multiaddr.Multiaddr, 0),
			}
			for _, addrString := range nodeInfo.Addr {
				addr, err := multiaddr.NewMultiaddr(addrString)
				if err != nil {
					continue
				}
				peerinfo.Addrs = append(peerinfo.Addrs, addr)
			}
			if len(peerinfo.Addrs) != 0 {
				peerInfos = append(peerInfos, peerinfo)
			}
		}
		return peerInfos, nil
	}
}
