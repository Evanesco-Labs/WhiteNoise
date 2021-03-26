package whitenoise

import (
	"encoding/binary"
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
