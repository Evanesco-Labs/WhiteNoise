package crypto

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"whitenoise/common/account"
)

func GenPrivKey() crypto.PrivKey {
	if acc := account.GetAccount(); acc != nil {
		return acc.GetPrivKey()
	}
	return nil
}