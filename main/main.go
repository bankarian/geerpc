package main

import (
	"fmt"
	"geerpc"
	"log"
	"net"
	"sync"
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


	client, _ := geerpc.Dial("tcp", <-addr)
	defer func() {
		client.Close()
	}()
	time.Sleep(time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		num := i
		go func() {
			defer wg.Done()
			var reply string
			if err := client.Call("Test.Hello", fmt.Sprintf("Hello %d", num), &reply); err != nil {
				log.Fatal("rpc call error")
			}
			log.Println("reply ", reply)
		}()
	}
	wg.Wait()
}
