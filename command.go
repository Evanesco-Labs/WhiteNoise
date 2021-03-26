package whitenoise

import (
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mr-tron/base58"
	"io"
	"whitenoise/log"
	"whitenoise/pb"
)

const CMD_PROTOCOL string = "cmd"

type CmdHandler struct {
	service *NetworkService
	CmdList map[string]bool
}

func (c *CmdHandler) run() {
}

//change to cmdHandler
func (s *NetworkService) CmdStreamHandler(stream network.Stream) {
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

	//dispatch
	var payload = pb.Command{}
	err = proto.Unmarshal(payloadBytes, &payload)
	if err != nil {
		log.Error("unmarshal err", err)

	}
	switch payload.Type {
	case pb.Cmdtype_SessionExPend:
		log.Info("Get session expend cmd")
		var cmd = pb.SessionExpend{}
		err = proto.Unmarshal(payload.Data, &cmd)
		if err != nil {
			break
		}
		ack := pb.Ack{
			CommandId: payload.CommandId,
			Result:    false,
			Data:      []byte{},
		}

		stream, err := s.host.NewStream(s.ctx, str.RemotePeer, core.ProtocolID(ACK_PROTOCOL))
		if err != nil {
			break
		}
		ackStream := NewStream(stream)

		session, ok := s.SessionMapper.SessionmapID[cmd.SessionId]
		if !ok {
			log.Warnf("No such session: %v", cmd.SessionId)
			ack.Data = []byte("No such session")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			encoded := EncodePayload(data)
			_, err = ackStream.RW.Write(encoded)
			if err != nil {
				break
			}
			err = ackStream.RW.Flush()
			if err != nil {
				break
			}
			break
		}
		if session.IsReady() {
			log.Warnf("Session is ready, ack as both joint and relay", cmd.SessionId)
			ack.Result = true
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			encoded := EncodePayload(data)
			_, err = ackStream.RW.Write(encoded)
			if err != nil {
				break
			}
			err = ackStream.RW.Flush()
			if err != nil {
				break
			}
			break
		}
		id, err := peer.Decode(cmd.PeerId)
		if err != nil {
			log.Warnf("Decode peerid %v err: %v", cmd.PeerId, err)
			ack.Data = []byte("Decode peerid error")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			encoded := EncodePayload(data)
			_, err = ackStream.RW.Write(encoded)
			if err != nil {
				break
			}
			err = ackStream.RW.Flush()
			if err != nil {
				break
			}
			break
		}
		err = s.NewSessionToPeer(id, cmd.SessionId, RelayRole, JointRole)
		if err != nil {
			log.Errorf("NewSessionToPeer err: %v", err)
			ack.Data = []byte("NewSessionToPeer error")
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			encoded := EncodePayload(data)
			_, err = ackStream.RW.Write(encoded)
			if err != nil {
				break
			}
			err = ackStream.RW.Flush()
			if err != nil {
				break
			}
			break
		} else {
			ack.Result = true
			data, err := proto.Marshal(&ack)
			if err != nil {
				break
			}
			encoded := EncodePayload(data)
			_, err = ackStream.RW.Write(encoded)
			if err != nil {
				break
			}
			err = ackStream.RW.Flush()
			if err != nil {
				break
			}
			break
		}

	case pb.Cmdtype_Disconnect:
		log.Info("get Disconnect cmd")
	}

}

func EncodeID(hash []byte) string {
	return base58.Encode(hash)
}
