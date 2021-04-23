package util

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
)

var DEBUG int

// addresses and message queues, for server and log collector
var Server_addr string
var Server_port string
var LC_addr string
var LC_port string

type MessageQueue struct {
	Req   map[int64]int
	Mutex sync.RWMutex
}

type RetrievalQueue struct {
	Req   map[int64]map[string]int
	Mutex sync.RWMutex
}

// tools

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

func IsFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func GetTime() int64 {
	now := time.Now()      // current local time
	nsec := now.UnixNano() // number of nanoseconds since January 1, 1970 UTC
	return nsec
}
