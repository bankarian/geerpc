package main

import (
	"fmt"
	"geerpc"
	"log"
	"net"
	"sync"
	"time"
)

type Foo string

type Arg struct {
	A, B int
}

type Reply map[string]string

func (foo *Foo) Sum(arg Arg, reply Reply) error {
	(reply)["expr"] = fmt.Sprintf("%d+%d=%d", arg.A, arg.B, arg.A+arg.B)
	return nil
}

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

	var foo Foo
	geerpc.Register(&foo)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		num := i
		go func() {
			defer wg.Done()
			var reply Reply
			arg := Arg{num, num + 5}
			if err := client.Call("Foo.Sum", arg, &reply); err != nil {
				log.Println("RPC Call error", err)
				return
			}
			log.Println("reply ", reply)
		}()
	}
	wg.Wait()
}
