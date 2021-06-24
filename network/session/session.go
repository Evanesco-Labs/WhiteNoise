package session

import (
	"context"
	"errors"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-msgio"
	"github.com/Evanesco-Labs/WhiteNoise/common"
)

const SessionIdNon string = "SessionIDNon"

type Session struct {
	Id   string
	Pair StreamPair
	Role common.SessionRole
}

type StreamPair []Stream

type Stream struct {
	StreamId   string
	RemotePeer core.PeerID
	RW         msgio.ReadWriteCloser
	raw        network.Stream
	cancel     context.CancelFunc
	Ctx        context.Context
}

func NewStream(s network.Stream, parentCtx context.Context) Stream {
	ctx, cancel := context.WithCancel(parentCtx)
	return Stream{
		StreamId:   s.ID(),
		RW:         msgio.Combine(msgio.NewVarintWriter(s), msgio.NewVarintReader(s)),
		RemotePeer: s.Conn().RemotePeer(),
		raw:        s,
		cancel:     cancel,
		Ctx:        ctx,
	}
}

func NewSession() Session {
	return Session{
		Id:   SessionIdNon,
		Pair: StreamPair{},
		Role: 0,
	}
}

func (s *Session) SetSessionID(sID string) {
	s.Id = sID
}

func (s *Session) AddStream(stream Stream) {
	if s.IsReady() {
		return
	}
	s.Pair = append(s.Pair, stream)
	for {
		if len(s.Pair) > 2 {
			s.Pair = s.Pair[1:]
		} else {
			break
		}
	}
}

func (s *Session) Has(streamID string) bool {
	for _, s := range s.Pair {
		if s.StreamId == streamID {
			return true
		}
	}
	return false
}

func (s *Session) GetPattern(streamID string) (Stream, error) {
	if len(s.Pair) != 2 {
		return Stream{}, errors.New("session not ready")
	}
	for i, stream := range s.Pair {
		if stream.StreamId == streamID {
			return s.Pair[i^1], nil
		}
	}
	return Stream{}, errors.New("no such stream")
}

func (s *Session) GetPair() StreamPair {
	return s.Pair
}

func (s *Session) IsReady() bool {
	return len(s.Pair) == 2
}

func (s *Stream) Close() {
	s.cancel()
	s.raw.Close()
}

type Probe struct {
	SessionId string
	Rand      []byte
}
