package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func doWrite(conn net.Conn, src []byte) error {
	for len(src) > 0 {
		written, err := conn.Write(src)
		if err != nil {
			return err
		}

		src = src[written:]
	}
	return nil
}

func writeLengthPrefixed(conn net.Conn, buf []byte) (err error) {
	sizebuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizebuf, uint32(len(buf)))
	err = doWrite(conn, sizebuf)
	if err != nil {
		return
	}
	err = doWrite(conn, buf)
	return
}

// TODO different main methods for different cmds..........
func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:5253")
	if err != nil {
		fmt.Println("Error connecting", err)
		return
	}

	for i := 0; i < 10; i++ {
		mybuffer := make([]byte, 20)
		for j := 0; j < 20; j++ {
			mybuffer[j] = byte(j)
		}
		fmt.Println("Writing buffer of length", len(mybuffer))
		err = writeLengthPrefixed(conn, mybuffer)
		if err != nil {
			fmt.Println("Error writing buffer", err)
		}
	}
}
