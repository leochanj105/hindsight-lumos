package util

import (
	"encoding/binary"
)

func Int32ToBytes(data int32) []byte {
	bytebuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytebuf, uint32(data))
	return bytebuf
}

func BytesToInt32(bys []byte) int32 {
	return int32(binary.LittleEndian.Uint32(bys))
}

func Int64ToBytes(data int64) []byte {
	bytebuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytebuf, uint64(data))
	return bytebuf
}

func BytesToInt64(bys []byte) int64 {
	return int64(binary.LittleEndian.Uint64(bys))
}
