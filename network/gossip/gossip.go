package gossip

import (
	"google.golang.org/protobuf/proto"
	"github.com/Evanesco-Labs/WhiteNoise/common/log"
	"github.com/Evanesco-Labs/WhiteNoise/internal/pb"
)

func (service *DHTService) GossipJoint(desHash string, negCypher []byte) error {
	encNeg := pb.EncryptedNeg{
		Des:    desHash,
		Cypher: negCypher,
	}

	data, err := proto.Marshal(&encNeg)
	if err != nil {
		return err
	}
	log.Debug("gossip neg cypher")
	err = service.NoisePublish(data)
	if err != nil {
		return err
	}
	return nil
}
