package whitenoise

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/multiformats/go-multiaddr"
	"time"
	"whitenoise/log"
	"whitenoise/pb"
)

const SetSessionTimeout time.Duration = time.Second
const ExpendSessionTimeout time.Duration = time.Second * 3
const RegisterProxyTimeout time.Duration = time.Second
const NewCircuitTimeout time.Duration = time.Second * 5

type NetworkService struct {
	host          host.Host
	ctx           context.Context
	discovery     Discovery
	SessionMapper SessionManager
	PubsubService *PubsubService
	AckManager    *AckManager
	proxyService  *ProxyService
	ProxyNode     core.PeerID
}

func (service *NetworkService) TryConnect(peer core.PeerAddrInfo) {
	err := service.host.Connect(service.ctx, peer)
	if err != nil {
		log.Errorf("connect to %v error: %v", peer.ID, err)
	} else {
	}
}

func NewService(ctx context.Context, host host.Host, cfg *NetworkConfig) (*NetworkService, error) {
	peerChan := initMDNS(ctx, host, cfg.RendezvousString)

	service := NetworkService{
		host: host,
		ctx:  ctx,
		discovery: Discovery{
			PeerMap:  make(map[core.PeerID]core.PeerAddrInfo),
			peerChan: peerChan,
			event:    make(chan core.PeerAddrInfo),
		},
		SessionMapper: NewSessionMapper(),
		AckManager:    NewAckManager(),
		proxyService:  NewProxyService(),
	}
	err := service.NewPubsubService()
	if err != nil {
		log.Error("New service err: ", err)
	}
	service.host.SetStreamHandler(protocol.ID(RelayProtocol), service.RelayStreamHandler)
	service.host.SetStreamHandler(protocol.ID(CMD_PROTOCOL), service.CmdStreamHandler)
	service.host.SetStreamHandler(protocol.ID(ACK_PROTOCOL), service.AckManager.AckStreamHandler)
	service.host.SetStreamHandler(protocol.ID(PROXY_PROTOCOL), service.ProxyStreamHandler)
	return &service, nil
}

func (service *NetworkService) Start() {
	log.Infof("start service %v\n", peer.Encode(service.host.ID()))
	go service.discovery.run()
	service.PubsubService.Start()
	go service.PubsubService.GossipHandler()
	go func() {
		for {
			event := <-service.discovery.event
			//log.Infof("discover %v", event.ID)
			//service.host.Peerstore().AddAddrs(event.ID, event.Addrs, time.Hour)
			service.TryConnect(event)
		}
	}()
}

func (service *NetworkService) NewRelayStream(peerID core.PeerID) (string, error) {
	stream, err := service.host.NewStream(service.ctx, peerID, protocol.ID(RelayProtocol))
	if err != nil {
		log.Infof("newstream to %v error: %v\n", peerID, err)
		return "", err
	}
	log.Info("gen new stream: ", stream.ID())
	s := NewStream(stream)
	service.SessionMapper.AddSessionNonid(s)
	go service.RelayInboundHandler(s)
	log.Infof("Connected to:%v \n", peerID)
	_, err = s.RW.Write(NewMsg())
	if err != nil {
		log.Error("write err", err)
		return "", err
	}
	err = s.RW.Flush()
	if err != nil {
		log.Error("flush err", err)
		return "", err
	}
	return s.StreamId, nil
}

func (service *NetworkService) NewSessionToPeer(peerID core.PeerID, sessionID string, myRole SessionRole, otherRole SessionRole) error {
	streamId, err := service.NewRelayStream(peerID)
	if err != nil {
		return err
	}
	err = service.SetSessionId(sessionID, streamId, myRole, otherRole)
	if err != nil {
		return err
	}
	return nil
}

func (service *NetworkService) SetSessionId(sessionID string, streamID string, myRole SessionRole, otherRole SessionRole) error {
	stream, ok := service.SessionMapper.StreamMap[streamID]
	if ok {
		s, ok := service.SessionMapper.SessionmapID[sessionID]
		if !ok {
			s = NewSession()
			s.SetSessionID(sessionID)
			s.Role = myRole
		}
		s.AddStream(stream)
		service.SessionMapper.AddSessionId(sessionID, s)
		log.Infof("session: %v\n", s)
	} else {
		return errors.New("no such stream:" + streamID)
	}

	data, id := NewSetSessionIDCommand(sessionID, otherRole)
	_, err := stream.RW.Write(data)
	if err != nil {
		log.Error("write err", err)
		return err
	}
	err = stream.RW.Flush()
	if err != nil {
		log.Error("flush err", err)
		return err
	}
	res := Task{
		Id:      id,
		channel: make(chan Result),
	}
	service.AckManager.TaskMap[id] = res

	defer delete(service.AckManager.TaskMap, id)
	select {
	case <-time.After(SetSessionTimeout):
		err := errors.New("timeout")
		return err
	case result := <-res.channel:
		if result.ok {
			return nil
		} else {
			return errors.New("cmd rejected: " + string(result.data))
		}
	}
}

func (service *NetworkService) SendRelay(sessionid string, data []byte) {
	session, ok := service.SessionMapper.SessionmapID[sessionid]
	if !ok {
		log.Info("SendRelay no such session")
	}
	payload := NewRelay(data, sessionid)
	for _, stream := range session.GetPair() {
		_, err := stream.RW.Write(payload)
		if err != nil {
			log.Error("write err", err)
			return
		}
		err = stream.RW.Flush()
		if err != nil {
			log.Error("flush err", err)
			return
		}
	}

}

func (service *NetworkService) ExpendSession(relay core.PeerID, joint core.PeerID, sessionId string) error {
	stream, err := service.host.NewStream(service.ctx, relay, protocol.ID(CMD_PROTOCOL))
	if err != nil {
		return err
	}
	s := NewStream(stream)
	cmd := pb.SessionExpend{
		SessionId: sessionId,
		PeerId:    joint.String(),
	}
	cmdData, err := proto.Marshal(&cmd)
	if err != nil {
		return err
	}
	payload := pb.Command{
		CommandId: "",
		Type:      pb.Cmdtype_SessionExPend,
		From:      service.host.ID().String(),
		Data:      cmdData,
	}
	payloadNoId, err := proto.Marshal(&payload)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payloadNoId)
	payload.CommandId = EncodeID(hash[:])

	data, err := proto.Marshal(&payload)
	if err != nil {
		return err
	}

	encode := EncodePayload(data)
	_, err = s.RW.Write(encode)
	if err != nil {
		return err
	}
	err = s.RW.Flush()
	if err != nil {
		return err
	}
	task := Task{
		Id:      payload.CommandId,
		channel: make(chan Result),
	}
	service.AckManager.TaskMap[payload.CommandId] = task
	defer delete(service.AckManager.TaskMap, payload.CommandId)
	select {
	case <-time.After(ExpendSessionTimeout):
		return errors.New("timeout")
	case result := <-task.channel:
		if result.ok {
			return nil
		} else {
			return errors.New("cmd rejected: " + string(result.data))
		}
	}
}

func (service *NetworkService) RegisterProxy(proxyId core.PeerID) error {
	streamRaw, err := service.host.NewStream(service.ctx, proxyId, protocol.ID(PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := NewStream(streamRaw)
	hash := sha256.Sum256([]byte(service.host.ID().String()))
	newProxy := pb.NewProxy{
		Time:   time.Hour.String(),
		IdHash: EncodeID(hash[:]),
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
	request.ReqId = EncodeID(hash[:])
	reqData, err := proto.Marshal(&request)

	_, err = stream.RW.Write(EncodePayload(reqData))
	if err != nil {
		return err
	}
	err = stream.RW.Flush()
	if err != nil {
		return err
	}

	resChan := service.AckManager.AddTask(request.ReqId)
	defer delete(service.AckManager.TaskMap, request.ReqId)
	select {
	case <-time.After(RegisterProxyTimeout):
		return errors.New("timeout")
	case result := <-resChan:
		if !result.ok {
			return errors.New("register rejected: " + string(result.data))
		}
	}
	service.ProxyNode = proxyId
	log.Infof("Register to proxy %v", proxyId)
	return nil
}

func (service *NetworkService) NewCircuit(des core.PeerID, sessionId string) error {
	if _, ok := service.SessionMapper.SessionmapID[sessionId]; ok {
		return errors.New("circuit with same sessionId already exist")
	}

	err := service.NewSessionToPeer(service.ProxyNode, sessionId, CallerRole, EntryRole)
	if err != nil {
		return err
	}

	streamRaw, err := service.host.NewStream(service.ctx, service.ProxyNode, protocol.ID(PROXY_PROTOCOL))
	if err != nil {
		return err
	}
	stream := NewStream(streamRaw)

	hashID := sha256.Sum256([]byte(des.String()))
	newCircuit := pb.NewCircuit{
		To:        EncodeID(hashID[:]),
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
	request.ReqId = EncodeID(hash[:])
	reqData, err := proto.Marshal(&request)

	_, err = stream.RW.Write(EncodePayload(reqData))
	if err != nil {
		return err
	}
	err = stream.RW.Flush()
	if err != nil {
		return err
	}

	resChan := service.AckManager.AddTask(request.ReqId)
	defer delete(service.AckManager.TaskMap, request.ReqId)
	select {
	case <-time.After(NewCircuitTimeout):
		return errors.New("timeout")
	case result := <-resChan:
		if !result.ok {
			return errors.New("new circuit rejected: " + string(result.data))
		}
	}
	return nil
}

func (service *NetworkService) GossipJoint(desHash string, join peer.ID, sessionId string) error {
	var neg = pb.Negotiate{
		Id:          "",
		Join:        join.String(),
		SessionId:   sessionId,
		Destination: desHash,
		Sig:         []byte{},
	}

	negNoID, err := proto.Marshal(&neg)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(negNoID)
	neg.Id = EncodeID(hash[:])
	data, err := proto.Marshal(&neg)
	if err != nil {
		return err
	}

	err = service.PubsubService.Publish(data)
	if err != nil {
		return err
	}
	return nil
}

func NewDummyHost(ctx context.Context, cfg *NetworkConfig) (host.Host, error) {
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.ListenHost, cfg.ListenPort))
	transport, err := noise.New(priv)
	if err != nil {
		return nil, err
	}
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Security(noise.ID, transport),
		libp2p.Identity(priv))
	if err != nil {
		return nil, err
	}
	return host, nil
}
