package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestMarshall(t *testing.T)  {
	priv,pub,_ := GenerateKeyPair(Ed25519,rand.Reader)
	fmt.Println(len(pub.GetWhiteNoiseID()))
	fmt.Println(len(priv.Bytes()))

	sk,pk,_ := GenerateKeyPair(ECDSA,rand.Reader)
	fmt.Println(len(pk.GetWhiteNoiseID()))
	fmt.Println(len(sk.Bytes()))
}
