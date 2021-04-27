package memory

import (
	"fmt"
	"os"
	"syscall"

	. "github.com/geraldleizhang/hindsight/agent/pkg/util"
)

type Pool struct {
	Pool []byte
}

type Dict struct {
	Dict []byte
}

var SharedPool Pool
var SharedDict Dict

var Cap int
var Buf_length int

func MemInit(fname string, size int) []byte {
	for {
		if IsFileExists(fname) == true {
			break
		}
	}
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("open file failed:", err)
	}
	fd := int(f.Fd())
	syscall.Ftruncate(fd, int64(size))

	mem, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)

	if err != nil {
		fmt.Println("mmap failed:", err)
	}

	return mem
}

func GetBufMetadata(buffer_id int) (int64, int64) {
	offset := buffer_id * Buf_length * 4
	request_id := BytesToInt64(SharedPool.Pool[offset : offset+8])
	timestamp := BytesToInt64(SharedPool.Pool[offset+8 : offset+16])
	return request_id, timestamp
}

func GetRequestID(buffer_id int) int64 {
	offset := buffer_id * Buf_length * 4
	request_id := BytesToInt64(SharedPool.Pool[offset : offset+8])
	return request_id
}

func GetRawRequestID(buffer_id int) []byte {
	offset := buffer_id * Buf_length * 4
	return SharedPool.Pool[offset : offset+8]
}

func GetBreadcrumbs(buffer_id int) []int {
	offset := buffer_id * Buf_length * 4
	breadcrumb_count := BytesToInt32(SharedPool.Pool[offset+32 : offset+36])
	if breadcrumb_count == 0 {
		return nil
	}
	var res []int
	for i := 0; i < int(breadcrumb_count); i++ {
		res = append(res, int(BytesToInt32(SharedPool.Pool[offset+36+i*4:offset+40+i*4])))
	}
	return res
}
