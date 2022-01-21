package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type Collector struct {
	port string
}

func (c *Collector) Init(port string) {
	c.port = port
}

func (c *Collector) Run(ctx context.Context) {
	fmt.Println("Collector listening on TCP port", c.port)
	listener, err := net.Listen("tcp", ":"+c.port)
	if err != nil {
		fmt.Println("Error listening on port", c.port, err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting new connection", err)
			return
		}
		go c.handleConnection(conn)
	}
}

func doRead(conn net.Conn, dst []byte) error {
	for len(dst) > 0 {
		read, err := conn.Read(dst)
		if err != nil {
			return err
		}

		dst = dst[read:]
	}
	return nil
}

func readLengthPrefixed(conn net.Conn) (buf []byte, err error) {
	szbuf := make([]byte, 4)
	err = doRead(conn, szbuf)
	if err != nil {
		return
	}
	sz := binary.LittleEndian.Uint32(szbuf)
	buf = make([]byte, sz)
	err = doRead(conn, buf)
	return
}

func (c *Collector) handleConnection(conn net.Conn) {
	next_print := time.Now()
	count := 0

	for {
		buf, err := readLengthPrefixed(conn)
		if err != nil {
			fmt.Println("Error in handleConnection receiving next buffer", err)
			return
		}
		memory.ExtractBufferHeader(buf)
		// header := memory.ExtractBufferHeader(buf)
		// fmt.Println("Read buf of size", len(buf), header)
		count += len(buf)

		if next_print.Before(time.Now()) {
			tput := float32(count) / float32(1024*1024)
			fmt.Printf("%.2f MB/s\n", tput)
			count = 0
			next_print = next_print.Add(time.Duration(1) * time.Second)
		}
	}
}
