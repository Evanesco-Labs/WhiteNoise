package whitenoise

import (
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"io"
	"time"
	"whitenoise/log"
	"whitenoise/pb"
)

const PROXY_PROTOCOL string = "proxyprotocol"
const PROXYSERIVCETIME time.Duration = time.Hour

type ProxyService struct {
	ClientMap map[string]clientInfo
}

type clientInfo struct {
	peerID core.PeerID
	state  int
	time   time.Duration
}

func NewProxyService() *ProxyService {
	return &ProxyService{ClientMap: make(map[string]clientInfo)}
}

func (service *NetworkService) ProxyStreamHandler(stream network.Stream) {
	defer stream.Close()
	str := NewStream(stream)
	lBytes := make([]byte, 4)
	_, err := io.ReadFull(str.RW, lBytes)
	if err != nil {
		return
	}

	l := Bytes2Int(lBytes)
	payloadBytes := make([]byte, l)
	_, err = io.ReadFull(str.RW, payloadBytes)
	if err != nil {
		log.Warn("payload not enough bytes")
		return
	}

	var request = pb.Request{}

	err = proto.Unmarshal(payloadBytes, &request)
	if err != nil {
		return
	}

	ackStreamRaw, err := service.host.NewStream(service.ctx, str.RemotePeer, core.ProtocolID(ACK_PROTOCOL))
	defer ackStreamRaw.Close()
	if err != nil {
		return
	}
	ackStream := NewStream(ackStreamRaw)

	var ack = pb.Ack{
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
			ack.Data = []byte("Unmarshal newProxy err")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			if err != nil {
				break
			}
			break
		}
		if _, ok := service.proxyService.ClientMap[newProxyReq.IdHash]; ok {
			ack.Data = []byte("Proxy already")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			if err != nil {
				break
			}
			break
		}

		duration, err := time.ParseDuration(newProxyReq.Time)
		if err != nil {
			duration = PROXYSERIVCETIME
		}
		service.proxyService.ClientMap[newProxyReq.IdHash] = clientInfo{
			peerID: str.RemotePeer,
			state:  1,
			time:   duration,
		}

		ack.Result = true
		data, _ := proto.Marshal(&ack)
		err = WriteOnce(data, ackStream)
		//todo:deal with err
		if err != nil {
			log.Error(err)
		}
		log.Infof("add new client %v", str.RemotePeer.String())
	case pb.Reqtype_NewCircuit:
		var newCircuit = pb.NewCircuit{}
		err = proto.Unmarshal(request.Data, &newCircuit)
		if err != nil {
			ack.Data = []byte("Unmarshal newCircuit err")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			if err != nil {
				break
			}
			break
		}

		hash := sha256.Sum256([]byte(str.RemotePeer.String()))
		if _, ok := service.proxyService.ClientMap[EncodeID(hash[:])]; !ok {
			ack.Data = []byte("Not registered client")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			if err != nil {
				break
			}
			break
		}

		if session, ok := service.SessionMapper.SessionmapID[newCircuit.SessionId]; !ok {
			ack.Data = []byte("No stream for this session yet")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			break
		} else if session.IsReady() {
			ack.Data = []byte("Session is full")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			break
		}

		//server and client connect to the same proxy
		//todo: this scheme needs review
		if clientInfo, ok := service.proxyService.ClientMap[newCircuit.To]; ok {
			log.Info("client server both to me")
			err = service.NewSessionToPeer(clientInfo.peerID, newCircuit.SessionId, ExitRole, AnswerRole)
			if err != nil {
				ack.Data = []byte(err.Error())
				data, err := proto.Marshal(&ack)
				if err != nil {
					log.Errorf("marshall err %v", err)
					break
				}
				err = WriteOnce(data, ackStream)
				break
			}
			ack.Result = true
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			if err != nil {
				break
			}
			break
		}

		//todo:add join node select scheme
		var join = core.PeerID("")
		for _,id := range service.host.Peerstore().Peers() {
			if id != service.host.ID() && id != str.RemotePeer {
				hash := sha256.Sum256([]byte(id.String()))
				if EncodeID(hash[:]) == newCircuit.To {
					continue
				}
				join = id
			}
		}

		err = service.NewSessionToPeer(join, newCircuit.SessionId, EntryRole, JointRole)
		if err != nil {
			ack.Data = []byte("NewSessionToPeer err" + err.Error())
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			break
		}

		err = service.GossipJoint(newCircuit.To, join, newCircuit.SessionId)
		if err != nil {
			ack.Data = []byte("GossipJoint err" + err.Error())
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			err = WriteOnce(data, ackStream)
			break
		}
		ack.Result = true
		data, err := proto.Marshal(&ack)
		if err != nil {
			break
		}
		err = WriteOnce(data, ackStream)
		if err != nil {
			break
		}
	}
}

func WriteOnce(payload []byte, stream Stream) error {
	encoded := EncodePayload(payload)
	_, err := stream.RW.Write(encoded)
	if err != nil {
		return err
	}
	err = stream.RW.Flush()
	if err != nil {
		return err
	}
	return nil
}
