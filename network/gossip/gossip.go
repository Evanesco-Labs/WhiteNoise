package gossip

import (
	"crypto/sha256"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/protobuf/proto"
	"whitenoise/internal/pb"
	"whitenoise/secure"
)

func (service *DHTService) GossipJoint(desHash string, join peer.ID, sessionId string) error {
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
	neg.Id = secure.EncodeID(hash[:])
	data, err := proto.Marshal(&neg)
	if err != nil {
		return err
	}
	err = service.NoisePublish(data)
	if err != nil {
		return err
	}
	return nil
}
