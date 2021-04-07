package secure

import (
	"encoding/binary"
	"github.com/mr-tron/base58"
	"io"
)

const TYPELEN = 4
const LENLEN = 4
const MaxLen = 1000

func EncodePayload(data []byte) []byte {
	l := uint32(len(data))
	b := Int2Bytes(l)
	return append(b[:], data...)
}

func Int2Bytes(l uint32) [LENLEN]byte {
	var res [LENLEN]byte
	binary.BigEndian.PutUint32(res[:], l)
	return res
}

func Bytes2Int(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func EncodeMSGIDHash(hash []byte) string {
	return base58.Encode(hash)
}

func ReadPayload(reader io.Reader) ([]byte, error) {
	lBytes := make([]byte, 4)
	_, err := io.ReadFull(reader, lBytes)
	if err != nil {
		return nil, err
	}

	l := Bytes2Int(lBytes)
	msgBytes := make([]byte, l)
	_, err = io.ReadFull(reader, msgBytes)
	if err != nil {
		return nil, err
	}
	return msgBytes, nil
}
