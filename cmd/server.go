package main

import (
	"context"
	"errors"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"whitenoise/common"
)

type Server struct {
}

func newServer() *Server {
	s := &Server{}
	return s
}

func (s Server) GenSessiontoPeer(ctx context.Context, cmd *GenSessionCmd) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	to, err := peer.Decode(cmd.PeerId)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}

	err = node.NoiseService.Relay().NewSessionToPeer(to, cmd.SessionId, common.SessionRole(cmd.MyRole), common.SessionRole(cmd.OtherRole))
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	res.Ok = true
	return &res, err
}

func (s Server) RegisterProxy(ctx context.Context, cmd *RegProxyCmd) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	proxy, err := peer.Decode(cmd.PeerId)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	err = node.NoiseService.RegisterProxy(proxy)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	res.Ok = true
	return &res, err
}

func (s Server) ExtendSession(ctx context.Context, cmd *ExtendCmd) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	relay, err := peer.Decode(cmd.Relay)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	join, err := peer.Decode(cmd.Joint)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	err = node.NoiseService.Cmd().ExpendSession(relay, join, cmd.SessionId)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	res.Ok = true
	return &res, err
}

func (s Server) SendRelayMsg(ctx context.Context, msg *RelayMsg) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	conn, ok := node.NoiseService.Relay().GetSecureConn(msg.SessionID)
	if !ok {
		res.Ok = false
		res.Data = []byte("conn not exist")
		return &res, nil
	}
	_, err := conn.Write(msg.Msg)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, nil
	}
	res.Ok = true
	return &res, nil
}

func (s Server) GenCircuit(ctx context.Context, cmd *GenCircuitCmd) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	des, err := peer.Decode(cmd.Des)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	err = node.NoiseService.NewCircuit(des, cmd.SessionId)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	res.Ok = true
	return &res, nil
}

func (s Server) GossipJoint(ctx context.Context, msg *GossipMsg) (*Res, error) {
	if node == nil {
		return nil, errors.New("nil net")
	}
	var res = Res{
		Ok:   false,
		Data: []byte{},
	}
	join, err := peer.Decode(msg.Join)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	err = node.DHTService.GossipJoint(msg.DesHash, join, msg.SessionId)
	if err != nil {
		res.Ok = false
		res.Data = []byte(err.Error())
		return &res, err
	}
	res.Ok = true
	return &res, nil
}

//todo:?
func (s Server) mustEmbedUnimplementedRpcServer() {

}
