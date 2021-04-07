package memory

import (
	"fmt"
	"os"
	"syscall"
)

func MemInit(fname string, size int) []byte {
	// queue_size := CONST_NUM*LEN + size*LEN*2

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
