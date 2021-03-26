package whitenoise

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"io"
	"os"
	"whitenoise/log"
	"whitenoise/pb"
)

const RelayProtocol string = "relay"

func (s *NetworkService) RelayStreamHandler(stream network.Stream) {
	fmt.Println("Got a new stream: ", stream.ID())
	str := NewStream(stream)
	s.SessionMapper.AddSessionNonid(str)
	go s.RelayInboundHandler(str)
}

func (service *NetworkService) RelayInboundHandler(s Stream) {
	for {
		lBytes := make([]byte, 4)
		_, err := io.ReadFull(s.RW, lBytes)
		if err != nil {
			continue
		}

		l := Bytes2Int(lBytes)
		msgBytes := make([]byte, l)
		_, err = io.ReadFull(s.RW, msgBytes)
		if err != nil {
			log.Info("payload not enough bytes")
			continue
		}

		var relay pb.Relay = pb.Relay{}
		err = proto.Unmarshal(msgBytes, &relay)
		if err != nil {
			log.Error("unmarshal err", err)
			continue
		}

		switch relay.Type {
		case pb.Relaytype_SetSessionId:
			err = service.handleSetSession(&relay, s)
			if err != nil {
				log.Error("handle set session command err ", err)
				continue
			}

		case pb.Relaytype_Data:
			err = service.handleRelayMsg(&relay, s, msgBytes)
			if err != nil {
				log.Error("handle relay message err ", err)
				continue
			}
		case pb.Relaytype_Disconnect:
		case pb.Relaytype_Ack:
		}
	}
}

func (service *NetworkService) handleSetSession(relay *pb.Relay, s Stream) error {

	//ack
	stream, err := service.host.NewStream(service.ctx, s.RemotePeer, core.ProtocolID(ACK_PROTOCOL))

	if err != nil {
		log.Errorf("handleSetSession new stream err %v", err)
		return err
	}
	resStream := NewStream(stream)
	var ack = pb.Ack{
		CommandId: relay.Id,
		Result:    false,
		Data:      []byte{},
	}

	var setSession pb.SetSessionIdMsg
	err = proto.Unmarshal(relay.Data, &setSession)
	if err != nil {
		ack.Data = []byte("setSessionIdMsg unmarshall err " + err.Error())
		data, err_ := proto.Marshal(&ack)
		if err_ != nil {
			return err
		}
		encoded := EncodePayload(data)
		_, err_ = resStream.RW.Write(encoded)
		if err_ != nil {
			log.Error("write err", err_)
			return err
		}
		err_ = resStream.RW.Flush()
		if err_ != nil {
			log.Error("flush err", err_)
			return err
		}
		return err
	}

	session, ok := service.SessionMapper.SessionmapID[setSession.SessionId]
	if !ok {
		session = NewSession()
		session.SetSessionID(setSession.SessionId)
		session.Role = SessionRole(setSession.Role)
	}
	session.AddStream(s)
	service.SessionMapper.AddSessionId(setSession.SessionId, session)
	log.Infof("add sessionid %v to stream %v\n", setSession.SessionId, s.StreamId)
	log.Infof("session: %v\n", session)

	ack.Result = true
	data, err := proto.Marshal(&ack)
	if err != nil {
		return err
	}
	encoded := EncodePayload(data)
	_, err = resStream.RW.Write(encoded)
	if err != nil {
		log.Error("write err", err)
		return err
	}
	err = resStream.RW.Flush()
	if err != nil {
		log.Error("flush err", err)
		return err
	}
	return nil
}

func (service *NetworkService) handleRelayMsg(relay *pb.Relay, s Stream, data []byte) error {
	var relayMsg pb.RelayMsg
	err := proto.Unmarshal(relay.Data, &relayMsg)
	if err != nil {
		return err
	}
	log.Info("Handling relay")
	session, ok := service.SessionMapper.SessionmapID[relayMsg.SessionId]
	if !ok {
		log.Warn("relay no such session")
		return nil
	}
	if session.Role == CallerRole || session.Role == AnswerRole {
		//TODO:give relay msg to client for relay node
		log.Infof("Got msg from circuit %v: %v\n", relayMsg.SessionId, string(relayMsg.Data))
		return nil
	}
	if session.IsReady() {
		part, err := session.GetPattern(s.StreamId)
		if err != nil {
			log.Error("get part err", err)
			return err
		}
		_, err = part.RW.Write(EncodePayload(data))
		if err != nil {
			log.Error("write err", err)
			return err
		}
		err = part.RW.Flush()
		if err != nil {
			log.Error("flush err", err)
			return err
		}
	}
	return nil
}

func NewSetSessionIDCommand(sessionID string, otherRole SessionRole) ([]byte, string) {
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
	relay.Id = EncodeID(hash[:])
	comd, _ := proto.Marshal(&relay)
	return EncodePayload(comd), relay.Id
}

func NewMsg() []byte {
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Wake,
		Data: []byte{},
	}
	relayNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(relayNoId)
	relay.Id = EncodeID(hash[:])
	relayBytes, _ := proto.Marshal(&relay)
	return EncodePayload(relayBytes)
}

func NewRelay(data []byte, sessionID string) []byte {
	relayMsg := pb.RelayMsg{
		SessionId: sessionID,
		Data:      data,
	}
	relayMsgData, _ := proto.Marshal(&relayMsg)
	relay := pb.Relay{
		Id:   "",
		Type: pb.Relaytype_Data,
		Data: relayMsgData,
	}
	relayNoId, _ := proto.Marshal(&relay)
	hash := sha256.Sum256(relayNoId)
	relay.Id = EncodeID(hash[:])
	relayBytes, _ := proto.Marshal(&relay)
	return EncodePayload(relayBytes)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}
