package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error", err)
	}
	log.Println("rpc server listens on", lis.Addr())
	addr <- lis.Addr().String()
	geerpc.Accept(lis)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	// client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		conn.Close()
	}()

	time.Sleep(time.Second)

	// send option
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)

	// send requests
	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Test.Hello",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("argv Seq=%d", h.Seq))

		// get response
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
